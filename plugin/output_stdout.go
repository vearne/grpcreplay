package plugin

import (
	"github.com/vearne/grpcreplay/protocol"
	"os"
)

type StdOutput struct {
	codec string
}

func NewStdOutput(codec string) *StdOutput {
	var o StdOutput
	o.codec = codec
	return &o
}

func (o *StdOutput) Close() error {
	return nil
}

func (o *StdOutput) Write(msg *protocol.Message) (err error) {
	var (
		data []byte
	)

	c := protocol.GetCodec(o.codec)
	data, err = c.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = os.Stderr.Write(data)
	if err != nil {
		return err
	}
	// make it more readable
	_, err = os.Stderr.Write([]byte{'\n', '\n'})
	if err != nil {
		return err
	}
	return nil
}
