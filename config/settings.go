// Package config 包含 grpcreplay 的配置管理相关功能。
// 该包定义了应用程序的配置结构和命令行参数解析器。
package config

import (
	"fmt"
	"strconv"
	"time"
)

// MultiStringOption 实现了可以接受多个值的字符串命令行参数。
// 它允许同一个参数名被多次指定，所有值都会被收集到一个切片中。
// 例如：--input-raw="127.0.0.1:8080" --input-raw="127.0.0.1:8081"
type MultiStringOption struct {
	Params *[]string // 指向存储所有参数值的切片的指针
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

// MultiIntOption 实现了可以接受多个值的整数命令行参数。
// 它允许同一个参数名被多次指定，所有值都会被收集到一个切片中。
type MultiIntOption struct {
	Params *[]int // 指向存储所有参数值的切片的指针
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

// AppSettings 是主配置结构体，包含了 grpcreplay 应用程序的所有配置选项。
// 该结构体的字段对应于命令行参数，包括输入源、输出目标、过滤器、限流器等配置。
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
	// multiple workers call services concurrently
	OutputGRPCWorkerNumber int `json:"output-grpc-worker-number"`

	// --- outputfile ---
	OutputFileDir []string `json:"output-file-directory"`
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	OutputFileMaxSize int `json:"output-file-max-size"`
	// MaxBackups is the maximum number of old log files to retain.
	OutputFileMaxBackups int `json:"output-file-max-backups"`
	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.
	OutputFileMaxAge int `json:"output-file-max-age"`

	// 	output RocketMQ
	OutputRocketMQNameServer []string `json:"output-rocketmq-name-server"`
	OutputRocketMQTopic      string   `json:"output-rocketmq-topic"`
	OutputRocketMQAccessKey  string   `json:"output-rocketmq-access-key"`
	OutputRocketMQSecretKey  string   `json:"output-rocketmq-secret-key"`

	// --- filter ---
	IncludeFilterMethodMatch string `json:"include-filter-method-match"`

	// --- rate limit ---
	// Query per second
	RateLimitQPS int `json:"rate-limit-qps"`

	// --- other ---
	Codec string `json:"codec"`

	RecordResponse bool `json:"record-response"`

	// file or directory
	ProtoFileStr string `json:"proto"`
	ProtoFiles   []string

	// If the output has been processed, the maximum time to wait for the input to be processed
	WaitDefaultDuration time.Duration
}
