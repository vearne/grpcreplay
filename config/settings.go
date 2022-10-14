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

	// --- input ---
	InputRAW []string `json:"input-raw"`

	// input-file-directory
	InputFileDir       []string `json:"input-file-directory"`
	InputFileReadDepth int      `json:"input-file-read-depth"`
	// --- output ---
	OutputStdout  bool     `json:"output-stdout"`
	OutputGRPC    []string `json:"output-grpc"`
	OutputFileDir []string `json:"output-file-directory"`

	OutputKafkaConfig plugin.OutputKafkaConfig
	// --- other ---
	Codec string `json:"codec"`
}
