package filter

import (
	"github.com/vearne/grpcreplay/protocol"
	slog "github.com/vearne/simplelog"
	"regexp"
)

type MethodMatchIncludeFilter struct {
	r *regexp.Regexp
}

func NewMethodMatchIncludeFilter(expr string) *MethodMatchIncludeFilter {
	var f MethodMatchIncludeFilter
	var err error
	f.r, err = regexp.Compile(expr)
	if err != nil {
		slog.Fatal("expr error:%v", err)
	}
	return &f
}

// Filter :If ok is true, it means that the message can pass
func (f *MethodMatchIncludeFilter) Filter(msg *protocol.Message) (*protocol.Message, bool) {
	if f.r.MatchString(msg.Method) {
		return msg, true
	}
	return nil, false
}
