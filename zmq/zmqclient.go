package zmq

import (
	"encoding/binary"
	"fmt"
	"time"
	"webserver-arm/log"

	"github.com/pebbe/zmq4"
)

const (
	ZMQ_SYSTEM_INFO_REQ_ENDPOINT = "tcp://127.0.0.1:9971"
	ZMQ_SYSTEM_INFO_SUB_ENDPOINT = "tcp://127.0.0.1:9972"
)

// DealerClient DEALER 模式客户端
type ZmqClient struct {
	context  *zmq4.Context
	socket   *zmq4.Socket
	addr     string
	clientID string // 自定义客户端身份
	Topic    string
}

func NewZmqClient(serverAddr, clientID string) (*ZmqClient, error) {
	ctx, err := zmq4.NewContext()
	if err != nil {
		return nil, fmt.Errorf("create zmq context failed: %w", err)
	}
	socket, err := ctx.NewSocket(zmq4.DEALER)
	if err != nil {
		return nil, fmt.Errorf("create zmq socket failed: %w", err)
	}

	err = socket.SetIdentity(clientID)
	if err != nil {
		return nil, fmt.Errorf("set client identity failed: %v", err)
	}

	if err = socket.SetRcvtimeo(60 * time.Second); err != nil {
		return nil, fmt.Errorf("set recv timeout failed: %w", err)
	}

	// 3. 连接到服务端
	if err := socket.Connect(serverAddr); err != nil {
		return nil, fmt.Errorf("connect to server %s failed: %w", serverAddr, err)
	}

	log.Infof("zmq client init success, connect to %s, clientID: %s", serverAddr, clientID)

	return &ZmqClient{
		context:  ctx,
		socket:   socket,
		addr:     serverAddr,
		clientID: clientID,
	}, nil
}

func NewZmqSub(serverAddr, clientID, topic string) (*ZmqClient, error) {
	ctx, err := zmq4.NewContext()
	if err != nil {
		return nil, fmt.Errorf("create zmq context failed: %w", err)
	}
	socket, err := ctx.NewSocket(zmq4.SUB)
	if err != nil {
		return nil, fmt.Errorf("create zmq socket failed: %w", err)
	}

	err = socket.SetIdentity(clientID)
	if err != nil {
		return nil, fmt.Errorf("set client identity failed: %v", err)
	}

	err = socket.SetSubscribe(topic)
	if err != nil {
		return nil, fmt.Errorf("set subscribe topic failed: %v", err)
	}

	// 3. 连接到服务端
	if err := socket.Connect(serverAddr); err != nil {
		return nil, fmt.Errorf("connect to server %s failed: %w", serverAddr, err)
	}

	log.Infof("zmq client init success, connect to %s, clientID: %s", serverAddr, clientID)

	return &ZmqClient{
		context:  ctx,
		socket:   socket,
		addr:     serverAddr,
		clientID: clientID,
		Topic:    topic,
	}, nil
}

// BuildTypeFrame 构建类型标识帧
// 输入：[]interface{} 数组（最多处理前16个元素）
// 输出：8字节的typeFrame，每个字节高低4位各存一个类型码：
//
//	string → 1，int(全整数类型) → 2，float(浮点类型) →3，[]byte(binary)→4，未知/空→0
func buildTypeFrame(input ...interface{}) []byte {
	const maxElements = 16 // 最多处理16个元素，对应64位（16*4）
	var bits uint64        // 核心存储变量，对标C++的uint64_t bits

	// 遍历每个元素（最多16个），计算类型码并写入bits
	for i := 0; i < maxElements; i++ {
		var typeCode uint64 // 类型码，默认0（空/未知类型）

		// 仅处理输入范围内的元素，超出部分保持0
		if i < len(input) {
			// 类型判断：覆盖所有常见数值类型+字符串+二进制
			switch input[i].(type) {
			case string:
				typeCode = 1 // string → 1
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				typeCode = 2 // 所有整数类型 → 2
			case float32, float64:
				typeCode = 3 // 所有浮点类型 → 3
			case []byte:
				typeCode = 4 // 二进制数组 → 4
			default:
				typeCode = 0 // 未知类型（如bool/struct等）→ 0
			}
		}

		// 确保类型码仅占4位（&0xF），避免溢出
		typeCode &= 0xF
		// 按位写入：第i个元素的类型码左移 (i*4) 位，然后合并到bits
		// 对标C++的 (t << ((i - 1) * 4))（C++ i从1开始，Go i从0开始，逻辑等价）
		bits |= typeCode << (i * 4)
	}

	typeFrame := make([]byte, 8)
	binary.BigEndian.PutUint64(typeFrame, bits)
	log.Info("typeFrame:", typeFrame, ", input:", input)
	return typeFrame
}

func (c *ZmqClient) Close() {
	c.socket.Close()
	c.context.Term()
}

func (c *ZmqClient) Send(frame ...string) error {
	frames := make([]interface{}, len(frame))
	for i, v := range frame {
		frames[i] = v
	}
	typeFrame := buildTypeFrame(frames...)

	sendFrame := make([]interface{}, 0)
	sendFrame = append(sendFrame, typeFrame)
	sendFrame = append(sendFrame, frames...)

	_, err := c.socket.SendMessage(sendFrame...)
	if err != nil {
		return fmt.Errorf("ZmqClient send failed:%s", err.Error())
	}

	return nil
}

func (c *ZmqClient) Recv() ([]string, error) {
	resp, err := c.socket.RecvMessage(0)
	if err != nil {
		return nil, fmt.Errorf("recv server response failed: %w", err)
	}

	return resp, nil
}

func (c *ZmqClient) SimpleGetMsg(frame ...string) ([]string, error) {
	err := c.Send(frame...)
	if err != nil {
		return nil, fmt.Errorf("send status to server failed: %w", err)
	}
	resp, err := c.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv server response failed: %w", err)
	}
	return resp[1:], nil
}

func (c *ZmqClient) SendAndLoopReceiveMsg(frame ...string) (<-chan []string, error) {
	// 发送请求
	if err := c.Send(frame...); err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	msgCh := make(chan []string)
	go func() {
		defer close(msgCh)
		for {
			resp, err := c.Recv()
			if err != nil {
				// 记录错误或触发关闭信号（根据需求决定是否继续）
				log.Errorf("recv error: %v", err)
				return
			}
			// 将收到的消息发送到 channel
			msgCh <- resp
		}
	}()
	return msgCh, nil
}
