package plugin

import (
	"github.com/vearne/grpcreplay/protocol"
	"os"
)

type StdOutput struct {
	codec protocol.Codec
}

func NewStdOutput(codec string) *StdOutput {
	var o StdOutput
	o.codec = protocol.GetCodec(codec)
	return &o
}

func (o *StdOutput) Close() error {
	return nil
}

func (o *StdOutput) Write(msg *protocol.Message) (err error) {
	var (
		data []byte
	)

	data, err = o.codec.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = os.Stderr.Write(data)
	if err != nil {
		return err
	}
	// make it more readable
	_, err = os.Stderr.Write([]byte{'\n', '\n'})
	return err
}
