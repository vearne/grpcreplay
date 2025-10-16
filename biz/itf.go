package biz

import (
	"io"

	"github.com/vearne/grpcreplay/protocol"
)

// PluginReader 定义了输入插件的接口。
// 实现该接口的插件可以从各种数据源读取 gRPC 消息，
// 如网络抓包、文件、消息队列等。
type PluginReader interface {
	io.Closer
	// Read 从数据源读取一个 gRPC 消息。
	// 返回值包含解析后的消息和可能的错误。
	// 如果数据源已经关闭或没有更多数据，应该返回适当的错误。
	Read() (msg *protocol.Message, err error)
}

// PluginWriter 定义了输出插件的接口。
// 实现该接口的插件可以将 gRPC 消息写入各种目标，
// 如控制台、文件、消息队列或直接转发到其他 gRPC 服务。
type PluginWriter interface {
	io.Closer
	// Write 将一个 gRPC 消息写入目标。
	// 参数 msg 是要写入的消息。
	// 返回值是可能的错误，如果写入成功则返回 nil。
	Write(msg *protocol.Message) (err error)
}

// Limiter 定义了限流器的接口。
// 实现该接口的组件可以控制消息处理的速率，
// 防止系统过载或资源耗尽。
type Limiter interface {
	// Allow 检查是否允许处理当前消息。
	// 返回 true 表示允许处理，false 表示应该跳过当前消息。
	// 该方法通常基于时间窗口、令牌桶等算法实现限流。
	Allow() bool
}
