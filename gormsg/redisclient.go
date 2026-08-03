package gormsg

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"webserver-arm/log"

	"github.com/gomodule/redigo/redis"
)

// type RedisClient struct {
// 	addr     string
// 	conns    *redis.Pool
// 	isMaster atomic.Bool
// }

type subInfo struct {
	cancel context.CancelFunc
	topics []string // maybe no use
	conn   redis.Conn
}

type RedisClient struct {
	addr     string
	conns    *redis.Pool
	isMaster atomic.Bool
	subsMux  sync.Mutex
	subs     map[string]*subInfo // subID -> subInfo
}

// func NewRedisClient(addr string) *RedisClient {
// 	return &RedisClient{
// 		addr: addr,
// 		conns: &redis.Pool{
// 			Dial: func() (redis.Conn, error) {
// 				return redis.Dial("tcp", addr)
// 			},
// 		},
// 	}

// }

func NewRedisClient(addr string) *RedisClient {
	return &RedisClient{
		addr: addr,
		conns: &redis.Pool{
			MaxIdle:     10,
			MaxActive:   100,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr,
					redis.DialConnectTimeout(5*time.Second),
					//redis.DialReadTimeout(10*time.Second), // not set to keep connection
					redis.DialWriteTimeout(5*time.Second),
				)
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if time.Since(t) < time.Minute {
					return nil
				}
				_, err := c.Do("PING")
				return err
			},
		},
		subs: make(map[string]*subInfo),
	}
}

func (rc *RedisClient) SetValid(ok bool) {
	rc.isMaster.Store(ok)
}

func (rc *RedisClient) Valid() bool {
	return rc.isMaster.Load()
}

func (rc *RedisClient) Publish(topic string, message []byte) error {
	conn := rc.conns.Get()
	defer conn.Close()

	_, err := conn.Do("PUBLISH", topic, message)
	return err
}

// func (rc *RedisClient) Subscribe(onMessage func(topic string, data []byte), topics ...string) {
// 	go func() {
// 		for {
// 			err := rc.subscribe(onMessage, topics...)
// 			if err != nil {
// 				fmt.Println("subscribe failed:", err)
// 			}
// 			time.Sleep(3 * time.Second)
// 			fmt.Println("resubscribe...")
// 		}
// 	}()
// }

// func (rc *RedisClient) subscribe(onMessage func(topic string, data []byte), topics ...string) error {
// 	conn := rc.conns.Get()
// 	defer conn.Close()

// 	psc := redis.PubSubConn{Conn: conn}
// 	err := psc.Subscribe(redis.Args{}.AddFlat(topics)...)
// 	if err != nil {
// 		return fmt.Errorf("Subscribe err:%w", err)
// 	}
// 	for {
// 		switch v := psc.Receive().(type) {
// 		case redis.Message:
// 			onMessage(v.Channel, v.Data)
// 		case redis.Subscription: // 订阅信息
// 			if v.Count == len(topics) {
// 				fmt.Println("subscribe OK. topics:", topics)
// 			} else {
// 				//return fmt.Errorf("subscribe count %d not match topics len %d", v.Count, len(topics))
// 				fmt.Println("subscribe count ", v.Count, " not match topics len", len(topics))
// 			}
// 		case error:
// 			return v
// 		}
// 	}
// }

// //////////////////////

func (rc *RedisClient) Subscribe(ctx context.Context, onMessage func(topic string, data []byte), topics ...string) (string, error) {
	subID := fmt.Sprintf("%s-%d", topics[0], time.Now().UnixNano())

	go rc.subscribe(ctx, subID, onMessage, topics...)

	log.Infof("Subscribe ActiveCount: %d, subID: %s", rc.conns.ActiveCount(), subID)
	return subID, nil
}

func (rc *RedisClient) subscribe(parentCtx context.Context, subID string, onMessage func(topic string, data []byte), topics ...string) {
	ctx, cancel := context.WithCancel(parentCtx)

	rc.subsMux.Lock()
	rc.subs[subID] = &subInfo{cancel: cancel, topics: topics}
	rc.subsMux.Unlock()

	defer func() {
		// clean
		rc.subsMux.Lock()
		delete(rc.subs, subID)
		rc.subsMux.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			//fmt.Printf("subscribe task [%s] calcelled\n", subID)
			log.Infof("subscribe task [%s] calcelled done", subID)
			return
		default:
			err := rc.doSubscribe(ctx, subID, onMessage, topics...)
			if err != nil {
				//fmt.Printf("subscribe task[%s] failed: %v\n", subID, err)
				log.Errorf("subscribe task[%s] failed: %v", subID, err)
			} else {
				//fmt.Printf("subscribe task[%s] done\n", subID)
				log.Infof("subscribe task[%s] done", subID)
				return
			}

			select {
			case <-time.After(3 * time.Second):
				//fmt.Printf("subscribe task[%s] start reconnect...\n", subID)
				log.Infof("subscribe task[%s] start reconnect...", subID)
			}
		}
	}
}

func (rc *RedisClient) doSubscribe(ctx context.Context, subID string, onMessage func(topic string, data []byte), topics ...string) error {
	conn := rc.conns.Get()
	defer func() {
		conn.Close()
		rc.subsMux.Lock()
		if info, ok := rc.subs[subID]; ok {
			info.conn = nil
		}
		rc.subsMux.Unlock()
	}()

	rc.subsMux.Lock()
	if subi, ok := rc.subs[subID]; ok {
		subi.conn = conn
	}
	rc.subsMux.Unlock()

	psc := redis.PubSubConn{Conn: conn}
	err := psc.Subscribe(redis.Args{}.AddFlat(topics)...)
	if err != nil {
		return fmt.Errorf("psc subscribe error: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("subscribe doSubscribe cancelled done")
			return nil
		default:
			switch v := psc.Receive().(type) {
			case redis.Message:
				onMessage(v.Channel, v.Data)
			case redis.Subscription:
				//fmt.Printf("subscribe status: %s %s, Count: %d, topics len:%d\n", v.Kind, v.Channel, v.Count, len(topics))
				log.Infof("subscribe status: %s %s, Count: %d, topics len:%d", v.Kind, v.Channel, v.Count, len(topics))
			case error:
				return v
			}
		}
	}
}

func (rc *RedisClient) Unsubscribe(subID string) error {
	rc.subsMux.Lock()
	defer rc.subsMux.Unlock()

	info, exists := rc.subs[subID]
	if !exists {
		return fmt.Errorf("Unsubscribe subscription not found")
	}

	// 1. stop
	info.cancel()

	// 2. unsubscribe
	if info.conn != nil && info.conn.Err() == nil {
		psc := redis.PubSubConn{Conn: info.conn}
		if err := psc.Unsubscribe(redis.Args{}.AddFlat(info.topics)...); err != nil {
			return fmt.Errorf("Unsubscribe task[%s] failed: %v", err.Error())
		}
	} else {
		return fmt.Errorf("Unsubscribe conn is closed")
	}

	// 2. unsubscribe
	// psc := redis.PubSubConn{Conn: info.conn}
	// if err := psc.Unsubscribe(redis.Args{}.AddFlat(info.topics)...); err != nil {
	// 	return fmt.Errorf("Unsubscribe task[%s] failed: %v", err.Error())
	// }

	return nil
}
