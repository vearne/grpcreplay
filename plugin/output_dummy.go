package plugin

import (
	"github.com/vearne/grpcreplay/model"
	"os"
)

var payloadSeparator = "\n🐵🙈🙉\n"
var payloadSeparatorAsBytes = []byte(payloadSeparator)

// DummyOutput used for debugging, prints all incoming requests
type DummyOutput struct {
}

// NewDummyOutput constructor for DummyOutput
func NewDummyOutput() (di *DummyOutput) {
	di = new(DummyOutput)

	return
}

// PluginWrite writes message to this plugin
func (i *DummyOutput) PluginWrite(msg *model.Message) (int, error) {
	var n, nn int
	var err error
	n, err = os.Stdout.Write(msg.Meta)
	nn, err = os.Stdout.Write(msg.Data)
	n += nn
	nn, err = os.Stdout.Write(payloadSeparatorAsBytes)
	n += nn

	return n, err
}

func (i *DummyOutput) String() string {
	return "Dummy Output"
}
