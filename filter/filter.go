package filter

import "github.com/vearne/grpcreplay/protocol"

type Filter interface {
	// Filter :If ok is true, it means that the message can pass
	Filter(msg *protocol.Message) (*protocol.Message, bool)
}

type FilterChain struct {
	includeFilters []Filter
	excludeFilters []Filter
}

func NewFilterChain() *FilterChain {
	var chain FilterChain
	chain.includeFilters = make([]Filter, 0)
	chain.excludeFilters = make([]Filter, 0)
	return &chain
}

func (c *FilterChain) AddIncludeFilter(f Filter) {
	c.includeFilters = append(c.includeFilters, f)
}

func (c *FilterChain) AddExcludeFilters(f Filter) {
	c.excludeFilters = append(c.excludeFilters, f)
}

func (c *FilterChain) Filter(msg *protocol.Message) (*protocol.Message, bool) {
	for _, f := range c.includeFilters {
		if _, ok := f.Filter(msg); !ok {
			return nil, false
		}
	}

	for _, f := range c.excludeFilters {
		if _, ok := f.Filter(msg); !ok {
			return nil, false
		}
	}
	return msg, true
}
