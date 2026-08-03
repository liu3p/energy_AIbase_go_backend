package gormsg

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"webserver-arm/log"
)

const (
	RedisPort = 5804
)

type ScdRedis struct {
	clients     []*RedisClient
	masterIndex atomic.Int32
	// add
	subsMux sync.RWMutex
	subs    map[string]*GlobalSubscription
}

type Rmsg struct {
	Seq     int64  `json:"seq"`
	Method  string `json:"method"`
	Payload string `json:"payload"`
}

func NewScdRedis() (*ScdRedis, error) {
	scdRedis := &ScdRedis{subs: make(map[string]*GlobalSubscription)}

	// addrs, mIndex, err := scdRedis.getAccessConn()
	// if err != nil {
	// 	return nil, fmt.Errorf("getAccessConn failed:%w", err)
	// }
	// for index, addr := range addrs {
	// 	client := NewRedisClient(addr)
	// 	client.SetValid(index == mIndex)
	// 	scdRedis.clients = append(scdRedis.clients, client)
	// }

	return scdRedis, nil
}

// func (sr *ScdRedis) getAccessConn() ([]string, int, error) {
// 	ips, mIndex, err := scdnet.GetClusterNodesIP()
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	address := make([]string, 0)
// 	for _, ip := range ips {
// 		address = append(address, fmt.Sprintf("%s:%d", ip, RedisPort))
// 	}

// 	return address, mIndex, nil
// }

// func (sr *ScdRedis) updateClientStatus() error {
// 	_, mIndex, err := scdnet.GetClusterNodesIP()
// 	if err != nil || mIndex < 0 {
// 		// all unavailable
// 		return fmt.Errorf("GetClusterNodesIP failed:%w", err)
// 	}

// 	sr.masterIndex.Store(int32(mIndex))
// 	return nil
// }

func (sr *ScdRedis) getValidClient() (*RedisClient, error) {
	// err := sr.updateClientStatus()
	// if err != nil {
	// 	return nil, err
	// }
	return sr.clients[sr.masterIndex.Load()], nil
}

func (sr *ScdRedis) Publish(topic string, message []byte) error {
	client, err := sr.getValidClient()
	if err != nil {
		return fmt.Errorf("no valid client:%w", err)
	}
	return client.Publish(topic, message)
}

// func (sr *ScdRedis) Subscribe(onMessage func(topic string, data []byte), topics ...string) {
// 	for _, client := range sr.clients {
// 		client.Subscribe(onMessage, topics...)
// 	}
// }

///////////////////////////////

type GlobalSubscription struct {
	Topics      []string
	OnMessage   func(topic string, data []byte)
	ClientSubs  map[int]string
	CancelFuncs map[int]context.CancelFunc
	Active      atomic.Bool
}

func (sr *ScdRedis) Subscribe(onMessage func(topic string, data []byte), topics ...string) (string, error) {
	if len(topics) == 0 {
		return "", errors.New("topics empty")
	}

	globalSubID := fmt.Sprintf("scdsub-%s-%d", topics[0], time.Now().UnixNano())

	// record subscription
	globalSub := &GlobalSubscription{
		Topics:      topics,
		OnMessage:   onMessage,
		ClientSubs:  make(map[int]string),
		CancelFuncs: make(map[int]context.CancelFunc),
		Active:      atomic.Bool{},
	}
	globalSub.Active.Store(true)

	for i, client := range sr.clients {
		ctx, cancel := context.WithCancel(context.Background())
		clientSubID, err := client.Subscribe(ctx, onMessage, topics...)
		if err != nil {
			log.Error("subsribe client:", i, " error:", err)
			cancel()
			continue
		}

		// record
		globalSub.ClientSubs[i] = clientSubID
		globalSub.CancelFuncs[i] = cancel
	}

	sr.subsMux.Lock()
	sr.subs[globalSubID] = globalSub
	sr.subsMux.Unlock()

	return globalSubID, nil
}

func (sr *ScdRedis) Unsubscribe(globalSubID string) error {
	sr.subsMux.Lock()
	defer sr.subsMux.Unlock()

	globalSub, exists := sr.subs[globalSubID]
	if !exists {
		return fmt.Errorf("global subscribe record not exist: %s", globalSubID)
	}

	for i, client := range sr.clients {
		if cancel, ok := globalSub.CancelFuncs[i]; ok {
			cancel()
		}

		if clientSubID, ok := globalSub.ClientSubs[i]; ok {
			// make sure to unsubscribe
			err := client.Unsubscribe(clientSubID)
			if err != nil {
				log.Error("client Unsubscribe failed:", err.Error())
			}
		}
	}

	delete(sr.subs, globalSubID)
	return nil
}

func (sr *ScdRedis) checkSubscriptions() {
	sr.subsMux.Lock()
	defer sr.subsMux.Unlock()

	for globalSubID, globalSub := range sr.subs {
		if !globalSub.Active.Load() {
			// skip not active subscription
			continue
		}

		// check each node's subscription status and re-subscribe if necessary
		for i, client := range sr.clients {
			if _, ok := globalSub.ClientSubs[i]; !ok {
				// still not subscribed, try to subscribe
				ctx, cancel := context.WithCancel(context.Background())
				clientSubID, err := client.Subscribe(ctx, globalSub.OnMessage, globalSub.Topics...)
				if err == nil {
					globalSub.ClientSubs[i] = clientSubID
					globalSub.CancelFuncs[i] = cancel
					fmt.Printf("recover global subsribe [%s] at %d \n", globalSubID, i)
				}
			}
		}
	}
}
