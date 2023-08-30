package config

import (
	"fmt"
	"github.com/vearne/grpcreplay/plugin"
	"strconv"
	"time"
)

type MultiStringOption struct {
	Params *[]string
}

func (h *MultiStringOption) String() string {
	if h.Params == nil {
		return ""
	}
	return fmt.Sprint(*h.Params)
}

// Set gets called multiple times for each flag with same name
func (h *MultiStringOption) Set(value string) error {
	if h.Params == nil {
		return nil
	}

	*h.Params = append(*h.Params, value)
	return nil
}

// MultiOption allows to specify multiple flags with same name and collects all values into array
type MultiIntOption struct {
	Params *[]int
}

func (h *MultiIntOption) String() string {
	if h.Params == nil {
		return ""
	}

	return fmt.Sprint(*h.Params)
}

// Set gets called multiple times for each flag with same name
func (h *MultiIntOption) Set(value string) error {
	if h.Params == nil {
		return nil
	}

	val, _ := strconv.Atoi(value)
	*h.Params = append(*h.Params, val)
	return nil
}

// AppSettings is the struct of main configuration
type AppSettings struct {
	ExitAfter time.Duration `json:"exit-after"`

	// ######################## input #######################
	InputRAW []string `json:"input-raw"`

	// --- input-file-directory ---
	InputFileDir         []string `json:"input-file-directory"`
	InputFileReadDepth   int      `json:"input-file-read-depth"`
	InputFileReplaySpeed float64  `json:"input-file-replay-speed"`

	// intput RocketMQ
	InputRocketMQNameServer []string `json:"input-rocketmq-name-server"`
	InputRocketMQTopic      string   `json:"input-rocketmq-topic"`
	InputRocketMQAccessKey  string   `json:"input-rocketmq-access-key"`
	InputRocketMQSecretKey  string   `json:"input-rocketmq-secret-key"`
	InputRocketMQGroupName  string   `json:"input-rocketmq-group-name"`

	// ######################## output ########################
	OutputStdout bool     `json:"output-stdout"`
	OutputGRPC   []string `json:"output-grpc"`

	// --- outputfile ---
	OutputFileDir []string `json:"output-file-directory"`
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	OutputFileMaxSize int `json:"output-file-max-size"`
	// MaxBackups is the maximum number of old log files to retain.
	OutputFileMaxBackups int `json:"output-file-max-backups"`
	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.
	OutputFileMaxAge  int `json:"output-file-max-age"`
	OutputKafkaConfig plugin.OutputKafkaConfig

	// 	output RocketMQ
	OutputRocketMQNameServer []string `json:"output-rocketmq-name-server"`
	OutputRocketMQTopic      string   `json:"output-rocketmq-topic"`
	OutputRocketMQAccessKey  string   `json:"output-rocketmq-access-key"`
	OutputRocketMQSecretKey  string   `json:"output-rocketmq-secret-key"`

	// --- filter ---
	IncludeFilterMethodMatch string `json:"include-filter-method-match"`
	// --- other ---
	Codec string `json:"codec"`
}
