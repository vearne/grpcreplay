package plugin

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/vearne/grpcreplay/protocol"
)

type RocketMQInput struct {
	pushConsumer rocketmq.PushConsumer
	topic        string
	msgChan      chan *primitive.MessageExt
}

func NewRocketMQInput(nameServers []string, topic, groupName string, accessKey, secretKey string) (*RocketMQInput, error) {
	var in RocketMQInput
	var err error

	in.topic = topic
	in.msgChan = make(chan *primitive.MessageExt, 1)
	if len(accessKey) > 0 {
		in.pushConsumer, err = rocketmq.NewPushConsumer(
			consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
			consumer.WithGroupName(groupName),
			consumer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
			consumer.WithCredentials(primitive.Credentials{
				AccessKey: accessKey,
				SecretKey: secretKey,
			}),
		)
	} else {
		in.pushConsumer, err = rocketmq.NewPushConsumer(
			consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
			consumer.WithGroupName(groupName),
			consumer.WithNsResolver(primitive.NewPassthroughResolver(nameServers)),
		)
	}
	if err != nil {
		return nil, err
	}

	selector := consumer.MessageSelector{Type: consumer.TAG, Expression: "*"}
	err = in.pushConsumer.Subscribe(topic, selector, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for i := range msgs {
			in.msgChan <- msgs[i]
		}
		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		return nil, err
	}

	err = in.pushConsumer.Start()
	return &in, err
}

func (in *RocketMQInput) Read() (*protocol.Message, error) {
	msgExt := <-in.msgChan
	var pm protocol.Message
	err := json.Unmarshal(msgExt.Body, &pm)
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (in *RocketMQInput) Close() error {
	return in.pushConsumer.Shutdown()
}
