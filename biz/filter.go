package biz

import (
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/filter"
)

func NewFilterChain(settings *config.AppSettings) (filter.Filter, error) {
	c := filter.NewFilterChain()
	c.AddExcludeFilters(filter.NewMethodExcludeFilter("grpc.reflection"))
	return c, nil
}
