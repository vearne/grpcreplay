package plugin

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
)

type RocketMQOutput struct {
	product rocketmq.Producer
	topic   string
}

func NewRocketMQOutput(nameServers []string, topic string) (*RocketMQOutput, error) {
	var o RocketMQOutput
	var err error
	o.topic = topic
	o.product, err = rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		producer.WithRetry(3),
	)
	if err != nil {
		return nil, err
	}
	err = o.product.Start()
	return &o, err
}

func (o *RocketMQOutput) close() error {
	return o.product.Shutdown()
}

func (o *RocketMQOutput) Write(msg *protocol.Message) (err error) {
	b, _ := json.Marshal(msg)
	pMsg := &primitive.Message{
		Topic: o.topic,
		Body:  b,
	}

	var result *primitive.SendResult
	result, err = o.product.SendSync(context.Background(), pMsg)
	if err != nil {
		slog.Error("RocketMQOutput-SendSync, error:%v", err)
	} else {
		slog.Debug("RocketMQOutput-SendSync, msgID:%v, status:%v",
			result.MsgID, result.Status)
	}
	return err
}
