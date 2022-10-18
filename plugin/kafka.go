package plugin

import (
	"github.com/Shopify/sarama"
)

// OutputKafkaConfig is the representation of kfka output configuration
// nolint: unused
type OutputKafkaConfig struct {
	producer   sarama.AsyncProducer
	Host       string `json:"output-kafka-host"`
	Topic      string `json:"output-kafka-topic"`
	UseJSON    bool   `json:"output-kafka-json-format"`
	SASLConfig SASLKafkaConfig
}

// SASLKafkaConfig SASL configuration
type SASLKafkaConfig struct {
	UseSASL   bool   `json:"input-kafka-use-sasl"`
	Mechanism string `json:"input-kafka-mechanism"`
	Username  string `json:"input-kafka-username"`
	Password  string `json:"input-kafka-password"`
}
