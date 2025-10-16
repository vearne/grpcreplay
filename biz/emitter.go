// Package biz 包含 grpcreplay 的核心业务逻辑，包括数据流处理、插件管理和消息分发等功能。
// 该包是整个系统的核心，负责协调输入插件、输出插件、过滤器和限流器之间的交互。
package biz

import (
	"io"
	"sync"

	"github.com/vearne/grpcreplay/filter"
	slog "github.com/vearne/simplelog"
)

// Emitter 表示一个用于管理插件通信的对象。
// 它负责协调输入插件和输出插件之间的数据流，并应用过滤器和限流器。
// Emitter 使用生产者-消费者模式，从输入插件读取数据，经过处理后发送到输出插件。
type Emitter struct {
	sync.WaitGroup
	plugins     *InOutPlugins // 输入和输出插件的集合
	filterChain filter.Filter // 过滤器链，用于过滤不需要的消息
	limiter     Limiter       // 限流器，用于控制消息处理速率
}

// NewEmitter 创建并初始化一个新的 Emitter 对象。
// 参数 f 是过滤器链，用于过滤消息；参数 lim 是限流器，用于控制处理速率。
// 返回初始化完成的 Emitter 实例。
func NewEmitter(f filter.Filter, lim Limiter) *Emitter {
	var e Emitter
	e.filterChain = f
	e.limiter = lim
	return &e
}

// Start 启动数据处理循环，将数据从输入插件发送到输出插件。
// 该方法为每个输入插件启动一个独立的 goroutine，实现并发处理。
// 每个 goroutine 会持续从输入插件读取数据，经过过滤和限流后发送到所有输出插件。
func (e *Emitter) Start(plugins *InOutPlugins) {
	e.plugins = plugins
	for _, in := range plugins.Inputs {
		e.Add(1)
		go func(in PluginReader) {
			defer e.Done()
			if err := e.CopyMulty(in, plugins.Outputs...); err != nil {
				slog.Debug("[EMITTER] error during copy: %q", err)
			}
		}(in)
	}
}

// Close 关闭所有 goroutine 并等待它们完成。
// 该方法会依次关闭所有插件（如果它们实现了 io.Closer 接口），
// 然后等待所有处理 goroutine 完成，确保优雅关闭。
func (e *Emitter) Close() {
	for _, p := range e.plugins.All {
		if cp, ok := p.(io.Closer); ok {
			cp.Close()
		}
	}
	if len(e.plugins.All) > 0 {
		// wait for everything to stop
		e.Wait()
	}
	e.plugins.All = nil // avoid Close to make changes again
}

// CopyMulty 从一个读取器复制数据到多个写入器。
// 该方法实现了数据流的核心逻辑：
// 1. 从源读取器读取消息
// 2. 通过过滤器链过滤消息
// 3. 应用限流策略
// 4. 将消息写入所有目标写入器
// 该方法会持续运行直到源读取器关闭或发生不可恢复的错误。
func (e *Emitter) CopyMulty(src PluginReader, writers ...PluginWriter) error {
	for {
		msg, err := src.Read()
		if err != nil {
			slog.Error("src.Read:%v", err)
			continue
		}
		msg, ok := e.filterChain.Filter(msg)
		if !ok {
			continue
		}

		if e.limiter != nil && !e.limiter.Allow() {
			continue
		}

		for _, dst := range writers {
			if err = dst.Write(msg); err != nil {
				slog.Error("dst.Write:%v", err)
			}
		}
	}
}
