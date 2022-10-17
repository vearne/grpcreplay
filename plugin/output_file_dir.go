package plugin

import (
	"github.com/pkg/errors"
	"github.com/vearne/grpcreplay/protocol"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

func IsValidDir(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return errors.Wrap(err, "invalid directory")
	}
	if !info.IsDir() {
		return errors.Errorf("%v is not direcotry", dirPath)
	}
	return nil
}

type FileDirOutputConfig struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	MaxSize int `json:"maxSize"`
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int `json:"maxBackups"`
	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.
	MaxAge int `json:"maxAge"`
}

type FileDirOutput struct {
	codec  protocol.Codec
	logger *lumberjack.Logger
}

func NewFileDirOutput(codec string, path string, cf *FileDirOutputConfig) *FileDirOutput {
	var ouput FileDirOutput
	ouput.codec = protocol.GetCodec(codec)
	ouput.logger = &lumberjack.Logger{
		Filename:   filepath.Join(path, "capture.log"),
		MaxSize:    cf.MaxSize, // megabytes
		MaxBackups: cf.MaxBackups,
		MaxAge:     cf.MaxAge, //days
		Compress:   true,      // disabled by default
	}
	return &ouput
}

func (o *FileDirOutput) Close() error {
	return o.logger.Close()
}

func (o *FileDirOutput) Write(msg *protocol.Message) (err error) {
	var (
		data []byte
	)

	data, err = o.codec.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = o.logger.Write(data)
	if err != nil {
		return err
	}
	_, err = o.logger.Write([]byte{'\n', '\n'})
	return err
}
