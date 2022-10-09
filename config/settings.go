package config

import (
	"fmt"
	"github.com/vearne/grpcreplay/plugin"
	"strconv"
	"time"
)

// MultiOption allows to specify multiple flags with same name and collects all values into array
type MultiOption struct {
	Params *[]string
}

func (h *MultiOption) String() string {
	if h.Params == nil {
		return ""
	}
	return fmt.Sprint(*h.Params)
}

// Set gets called multiple times for each flag with same name
func (h *MultiOption) Set(value string) error {
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

	InputFile          []string `json:"input-file"`
	InputFileLoop      bool     `json:"input-file-loop"`
	InputFileReadDepth int      `json:"input-file-read-depth"`
	// --- output ---
	OutputStdout bool     `json:"output-stdout"`
	OutputGRPC   []string `json:"output-grpc"`
	OutputFile   []string `json:"output-file"`
	//OutputFileConfig plugin.FileOutputConfig

	OutputKafkaConfig plugin.OutputKafkaConfig
}
