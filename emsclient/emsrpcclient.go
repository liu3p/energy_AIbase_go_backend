package emsclient

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"webserver-arm/log"
	"webserver-arm/models"
)

// var instance *EmsClient

// 定义消息类型常量
const (
	BodyLenIndex   = 0 // body长度，占四字节
	MsgSeqIndex    = 4 // 消息序列号，占四字节
	MsgTypeIndex   = 8 // 消息类型，占一字节
	MsgMethodIndex = 9 // 消息方法，占四字节

	HeadLength = 13 // 头部长度

	MessageTypeCall  = 10 // 调用
	MessageTypeReply = 11 // 回复
	MessageTypePush  = 12 // 推送（无需回复）
	MessageTypeError = 13 // 错误

)

var ErrShutdown = errors.New("connection is shut down")
var ErrNotMessageTypeReply = errors.New("messageType invalid")
var ErrErrorMessage = errors.New("received error")
var ErrUnknownMessageType = errors.New("unknown messageType")

type IEmsRPCReq interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type EmsRPCClient struct {
	conn     net.Conn
	shutDown bool
}

func (c *EmsRPCClient) send(method, seq uint32, body []byte) error {
	startTime := time.Now()
	if c.shutDown {
		return ErrShutdown
	}
	// 计算消息体长度
	bodyLength := uint32(len(body))
	// 构造消息
	message := make([]byte, HeadLength+int(bodyLength))
	binary.BigEndian.PutUint32(message[BodyLenIndex:MsgSeqIndex], bodyLength)
	binary.BigEndian.PutUint32(message[MsgSeqIndex:MsgTypeIndex], seq)
	message[MsgTypeIndex] = MessageTypeCall
	binary.BigEndian.PutUint32(message[MsgMethodIndex:HeadLength], method)
	copy(message[HeadLength:], body)
	// 发送消息
	_, err := c.conn.Write(message)
	log.Debug("send>> bodyLength:", bodyLength, ",msgSeq:", seq, ",msgType:", MessageTypeCall, "method:", method,
		",offset:", time.Since(startTime))
	return err
}

func (c *EmsRPCClient) recv() (method, msgSeq uint32, msgType uint8, body []byte, err error) {
	startTime := time.Now()
	header := make([]byte, HeadLength)
	err = c.read(header)
	if err != nil {
		return method, msgSeq, msgType, body, err
	}
	//var messageTypeError error
	// 解析消息头部
	bodyLength := binary.BigEndian.Uint32(header[BodyLenIndex:MsgSeqIndex])
	msgSeq = binary.BigEndian.Uint32(header[MsgSeqIndex:MsgTypeIndex])
	msgType = header[MsgTypeIndex]
	method = binary.BigEndian.Uint32(header[MsgMethodIndex:HeadLength])

	// if msgType != MessageTypeReply {
	// 	// 尽管头部消息类型出错，还是要读一下消息体
	// 	if msgType == MessageTypeError {
	// 		messageTypeError = ErrErrorMessage
	// 	} else {
	// 		messageTypeError = ErrNotMessageTypeReply
	// 	}
	// }
	// 读取消息体
	log.Debug("recv<< bodyLength:", bodyLength, ",msgSeq:", msgSeq, ",msgType:", msgType, "method:", method,
		",offset:", time.Since(startTime))
	body = make([]byte, bodyLength)
	err = c.read(body)
	if err != nil {
		return method, msgSeq, msgType, body, err
	}
	log.Debug("recv<< body over. msgSeq:", msgSeq, "over:", time.Since(startTime))

	return method, msgSeq, msgType, body, err
}

func (c *EmsRPCClient) read(buf []byte) error {
	currentLen := 0
	for currentLen < len(buf) {
		readLen, err := c.conn.Read(buf[currentLen:])
		if err != nil {
			return err
		}
		currentLen += readLen
	}

	return nil
}

func (c *EmsRPCClient) Dial(address string) error {
	var err error
	c.conn, err = net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.shutDown = false
	return nil
}
func (c *EmsRPCClient) Close() error {
	c.shutDown = true
	err := c.conn.Close()
	return err
}

//nolint:govet
type Call struct {
	CallMethod  uint32             // The name of the service and method to call.
	Reply       models.IEmsRPCResp // The reply from the function (*struct).
	Error       error
	Done        chan *Call                    // Receives *Call when Go is complete.
	PushHandler func(resp models.IEmsRPCResp) // push msg handle
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here. It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Error("ERROR: discarding Call reply due to insufficient Done chan capacity")
	}
}

const (
	NotifyReconn = 1
	NotifyStop   = 2
	MaxSeqValue  = 4294967290
	InitSeqValue = 10000 // start from 10000, for aviod conflict to method-id because push msg seq is same to method-id
)

//nolint:govet
type EmsRPCClientBase struct {
	serverAddr  string
	client      *EmsRPCClient
	stopConnect atomic.Bool
	recvSync    sync.WaitGroup

	seqInit sync.Once
	seq     uint32
	waiting sync.Map

	timeout time.Duration

	RunOK atomic.Bool
}

func (e *EmsRPCClientBase) getSeq() uint32 {
	e.seqInit.Do(func() {
		e.seq = InitSeqValue
	})
	result := atomic.AddUint32(&e.seq, 1)
	if result == math.MaxUint32 {
		atomic.StoreUint32(&e.seq, InitSeqValue)
		return InitSeqValue
	}
	return result
}

func (e *EmsRPCClientBase) waitingReply() {
	var err error
	for err == nil {
		method, seq, msgType, bufRecv, errRecv := e.client.recv()
		if errRecv != nil {
			err = fmt.Errorf("received error:%s", errRecv.Error())
			continue
			// switch errRecv {
			// case ErrNotMessageTypeReply, ErrErrorMessage:
			// 	// we need to try to stop the caller
			// default:

			// 	continue
			// }
		}

		value, exist := e.waiting.LoadAndDelete(seq)
		if !exist || value == nil {
			log.Errorf("recv invalid response, not found in waiting list. method: %d, seq: %d", method, seq)
			continue
		}
		retcall, ok := value.(*Call)
		if !ok {
			// we may not here
			err = fmt.Errorf("value not Call. method: %d, seq: %d", method, seq)
			continue
		}

		//var retcall *Call
		switch msgType {
		case MessageTypeCall:
			err = fmt.Errorf("waitingReply recv invalid messageType: %d", MessageTypeCall)
			continue

		case MessageTypePush, MessageTypeReply:
			errUnmarshal := retcall.Reply.Unmarshal(bufRecv)
			if errUnmarshal != nil {
				retcall.Error = fmt.Errorf("unmarshal received body failed: %s. method: %d, seq: %d, bufRecv:%s",
					errUnmarshal.Error(), method, seq, string(bufRecv))
			}
			if msgType == MessageTypePush {
				if retcall.Error == nil {
					go retcall.PushHandler(retcall.Reply.Clone())
				} else {
					log.Error("handle Push err:", retcall.Error)
				}
				// 继续接受消息
				retcall.Error = nil
				e.waiting.Store(seq, retcall)
				continue
			} else {
				retcall.done()
			}

		case MessageTypeError:
			retcall.Error = fmt.Errorf("SRVERR: %s", string(bufRecv))
			retcall.done()

		default:
			retcall.Error = ErrNotMessageTypeReply
			retcall.done()
			continue
		}
	}
	log.Warnf("connect recv shutdown. err: %s", err.Error())
	// Terminate pending calls.
	e.client.Close()
	if errors.Is(err, io.EOF) {
		err = ErrShutdown
	}
	// notify all calling function to shutdown
	e.waiting.Range(func(key, value any) bool {
		call, ok := value.(*Call)
		if !ok {
			// we may not here
			err = errors.New("range waiting, value not Call")
			return true
		}
		call.Error = err
		call.done()
		return true
	})

	//e.reconnNoify <- true
	e.recvSync.Done()
}

func (e *EmsRPCClientBase) RegistPushHandler(method uint32, resp models.IEmsRPCResp,
	handler func(resp models.IEmsRPCResp)) {
	seq := method
	call := &Call{
		CallMethod: method,
		Reply:      resp,
		//Done:       make(chan *Call, 1024), // hold some message
		PushHandler: handler,
	}
	e.waiting.Store(seq, call)
}

func (e *EmsRPCClientBase) Call(method uint32, req IEmsRPCReq, resp models.IEmsRPCResp) error {
	sendContect, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("request Marshal failed: %s", err.Error())
	}

	seq := e.getSeq()
	call := &Call{
		CallMethod: method,
		Reply:      resp,
		Done:       make(chan *Call, 1),
	}
	e.waiting.Store(seq, call)

	err = e.client.send(method, seq, sendContect)
	if err != nil {
		value, exist := e.waiting.LoadAndDelete(seq)
		if !exist || value == nil {
			log.Errorf("send err, not found in waiting list. method: %d, seq: %d", method, seq)
		}
		call, ok := value.(*Call)
		if !ok {
			// we may not here
			err = fmt.Errorf("value not Call. method: %d, seq: %d", method, seq)
		}
		call.Error = fmt.Errorf("send failed: %s. method: %d, seq: %d", err.Error(), method, seq)
		call.done()
	}
	log.Debug("")
	select {
	case call := <-call.Done:
		log.Debug("get call done. seq:", seq, " method:", call.CallMethod)
		return call.Error
	case <-time.After(e.timeout):
		return fmt.Errorf("request timeout. method: %d, seq: %d, timeout:%d", method, seq, e.timeout)
	}
}

func (e *EmsRPCClientBase) Connect() error {
	go func() {
		for !e.stopConnect.Load() {
			err := e.client.Dial(e.serverAddr)
			if err != nil {
				log.Warn("connect gateway failed. err:", err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			log.Info("connected to server:", e.serverAddr)
			e.recvSync.Add(1)
			go e.waitingReply()
			e.RunOK.Store(true)
			e.recvSync.Wait()
			e.RunOK.Store(false)
		}
		log.Warn("Connect goruntine exited!")
	}()

	return nil
}

func (e *EmsRPCClientBase) ConnectOnce() error {
	err := e.client.Dial(e.serverAddr)
	if err != nil {
		return err
	}
	e.recvSync.Add(1)
	go e.waitingReply()
	return nil
}

func (e *EmsRPCClientBase) Stop() {
	e.stopConnect.Store(true)
	e.RunOK.Store(false)
	e.client.Close()
}

func (e *EmsRPCClientBase) GetServerIP() string {
	return e.serverAddr
}

//nolint:govet
type EmsClient struct {
	EmsRPCClientBase
	mgr *Manager
}

func NewEmsClient(mgr *Manager, serverAddr string, timeoutSecond int) *EmsClient {
	defaultSecond := 10
	if timeoutSecond != 0 {
		defaultSecond = timeoutSecond
	}

	timeout := time.Duration(defaultSecond) * time.Second
	client := &EmsClient{
		EmsRPCClientBase: EmsRPCClientBase{
			serverAddr: serverAddr,
			client:     &EmsRPCClient{shutDown: true},
			//reconnNoify: make(chan bool),
			seq:     1,
			timeout: timeout,
		},
		mgr: mgr,
	}
	client.stopConnect.Store(false)
	client.RunOK.Store(false)
	return client
}

// func GetInstance() *EmsClient {
// 	once.Do(func() {
// 		instance = EmsClient("192.168.2.35:8676", 10)
// 	})
// 	return instance
// }
