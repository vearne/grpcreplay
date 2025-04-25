package filter

import (
	"github.com/vearne/grpcreplay/protocol"
	"strings"
)

type MethodExcludeFilter struct {
	exclude string
}

func NewMethodExcludeFilter(execlude string) *MethodExcludeFilter {
	var f MethodExcludeFilter
	f.exclude = execlude
	return &f
}

// Filter :If ok is true, it means that the message can pass
func (f *MethodExcludeFilter) Filter(msg *protocol.Message) (*protocol.Message, bool) {
	if strings.Contains(msg.Method, f.exclude) {
		return nil, false
	}
	return msg, true
}
