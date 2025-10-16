# grpcreplay 架构设计文档

## 📋 概述

grpcreplay 是一个用于捕获、记录和重放 gRPC 流量的网络监控工具。它采用插件化架构，支持多种输入源和输出目标，可用于灰度测试、压力测试和流量分析。

## 🏗️ 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        grpcreplay 架构图                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Input     │    │   Filter    │    │   Output    │         │
│  │  Plugins    │───▶│   Chain     │───▶│  Plugins    │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                   │                   │              │
│         ▼                   ▼                   ▼              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │ RAW Socket  │    │ Method      │    │  Console    │         │
│  │ File Reader │    │ Filter      │    │  File       │         │
│  │ RocketMQ    │    │ Rate Limit  │    │  gRPC       │         │
│  │ Consumer    │    │             │    │  RocketMQ   │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 🧩 核心组件

### 1. 数据流处理层 (biz/)

#### Emitter (数据分发器)
- **职责**: 协调输入插件、过滤器和输出插件之间的数据流
- **特点**: 
  - 支持一对多的数据分发
  - 并发处理多个输入源
  - 实现生产者-消费者模式

```go
type Emitter struct {
    sync.WaitGroup
    plugins     *InOutPlugins
    filterChain filter.Filter
    limiter     Limiter
}
```

#### 插件接口定义
```go
// 输入插件接口
type PluginReader interface {
    io.Closer
    Read() (msg *protocol.Message, err error)
}

// 输出插件接口  
type PluginWriter interface {
    io.Closer
    Write(msg *protocol.Message) (err error)
}

// 限流器接口
type Limiter interface {
    Allow() bool
}
```

### 2. HTTP/2 协议处理层 (http2/)

#### 核心功能
- **数据包捕获**: 使用 libpcap 捕获网络数据包
- **HTTP/2 解析**: 解析 HTTP/2 帧结构
- **HPACK 解压**: 处理 HTTP/2 头部压缩
- **gRPC 消息重建**: 从 HTTP/2 数据流重建完整的 gRPC 消息

#### 关键组件

**TCPBuffer (TCP缓冲区)**
```go
type TCPBuffer struct {
    expectedSeq uint32
    List        *skiplist.SkipList
    size        atomic.Int64
    // ...
}
```
- 处理 TCP 数据包的乱序和重复
- 使用滑动窗口算法管理数据包
- 支持并发安全的数据读取

**Http2Conn (HTTP/2连接)**
```go
type Http2Conn struct {
    Input       *ConnItem
    Output      *ConnItem
    DirectConn  *DirectConn
    Processor   *Processor
    // ...
}
```
- 管理 HTTP/2 连接的输入输出流
- 处理 HTTP/2 帧的解析和重组
- 维护连接状态和流状态

**PBFinder (Protobuf定义查找器)**
- 支持两种模式：
  - **反射模式**: 通过 gRPC 反射获取服务定义
  - **文件模式**: 从本地 .proto 文件加载定义

### 3. 插件系统 (plugin/)

#### 输入插件

**RAWInput (网络抓包)**
```go
type RAWInput struct {
    port      int
    ipSet     *util.StringSet
    connSet   *ConnSet
    // ...
}
```
- 使用 RAW Socket 捕获网络流量
- 实现 TCP 连接重置机制
- 支持多网卡监听

**FileDirInput (文件读取)**
- 从文件目录读取已保存的 gRPC 消息
- 支持压缩文件格式
- 可配置重放速度

**RocketMQInput (消息队列)**
- 从 RocketMQ 消费 gRPC 消息
- 支持消费者组和认证

#### 输出插件

**StdOutput (控制台输出)**
- 将 gRPC 消息格式化输出到控制台
- 支持多种编码格式 (simple/json)

**FileDirOutput (文件输出)**
- 将 gRPC 消息保存到文件
- 支持文件轮转和压缩
- 可配置文件大小和保留策略

**GRPCOutput (gRPC转发)**
- 将捕获的 gRPC 消息转发到目标服务
- 支持并发工作者
- 实现请求重放功能

**RocketMQOutput (消息队列输出)**
- 将 gRPC 消息发送到 RocketMQ
- 支持生产者认证

### 4. 过滤系统 (filter/)

#### 过滤器接口
```go
type Filter interface {
    Filter(msg *protocol.Message) (*protocol.Message, bool)
}
```

#### 内置过滤器
- **IncludeFilter**: 基于正则表达式的方法名过滤
- **ExcludeFilter**: 排除特定方法的过滤器
- **FilterChain**: 过滤器链，支持多个过滤器组合

### 5. 协议处理层 (protocol/)

#### Message 结构
```go
type Message struct {
    Meta     *Meta
    Method   string
    Request  *RequestResponse
    Response *RequestResponse
}
```

#### 编码器
- **SimpleCodec**: 简单文本格式编码
- **JSONCodec**: JSON 格式编码
- 支持自定义编码器扩展

### 6. 配置管理 (config/)

#### AppSettings
包含所有应用配置选项：
- 输入源配置 (RAW、文件、RocketMQ)
- 输出目标配置 (控制台、文件、gRPC、RocketMQ)
- 过滤器配置
- 限流配置
- 协议配置

## 🔄 数据流处理流程

### 1. 网络抓包模式

```
网络数据包 → RAW Socket → TCP重组 → HTTP/2解析 → gRPC消息重建 → 过滤器 → 限流器 → 输出插件
```

**详细步骤:**

1. **数据包捕获**
   - 使用 libpcap 在指定端口捕获数据包
   - 过滤出 TCP 协议的数据包

2. **TCP 连接管理**
   - 识别新的 TCP 连接
   - 使用 TCP Kill 机制重置旧连接
   - 强制客户端建立新连接

3. **HTTP/2 处理**
   - 检测 HTTP/2 连接前导码
   - 解析 HTTP/2 帧结构
   - 处理 HPACK 头部压缩

4. **gRPC 消息重建**
   - 从 HTTP/2 数据帧提取 gRPC 消息
   - 使用 Protobuf 定义解析消息体
   - 重建完整的请求/响应对

5. **消息处理**
   - 应用过滤器链
   - 执行限流策略
   - 分发到所有输出插件

### 2. 文件重放模式

```
文件读取 → 消息解析 → 速度控制 → 过滤器 → 限流器 → 输出插件
```

### 3. 消息队列模式

```
RocketMQ消费 → 消息解析 → 过滤器 → 限流器 → 输出插件
```

## 🔧 关键技术实现

### 1. TCP 连接重置机制

由于 gRPC 使用 HTTP/2 的 HPACK 压缩，需要从连接开始捕获才能正确解析头部。grpcreplay 使用类似 tcpkill 的机制：

1. 发现目标端口的所有活跃连接
2. 发送 RST 包重置这些连接
3. 强制客户端重新建立连接
4. 从新连接开始捕获和解析

### 2. HTTP/2 状态机

```go
type StateMachine struct {
    currentState State
    transitions  map[StateTransition]State
}
```

状态包括：
- `WaitPreface`: 等待 HTTP/2 前导码
- `WaitSetting`: 等待设置帧
- `Connected`: 连接已建立，可以处理数据

### 3. 并发安全设计

- **TCPBuffer**: 使用原子操作和跳表实现并发安全
- **ConnSet**: 使用读写锁保护连接集合
- **Emitter**: 使用 WaitGroup 管理 goroutine 生命周期

### 4. 内存管理

- **BufferPool**: 复用缓冲区减少 GC 压力
- **GoroutineSafeBuffer**: 线程安全的缓冲区实现
- 及时释放不再使用的资源

## 🚀 性能优化

### 1. 网络处理优化
- 使用 RAW Socket 减少内核态/用户态切换
- 批量处理数据包
- 零拷贝数据传输

### 2. 内存优化
- 对象池复用
- 流式处理大文件
- 及时释放资源

### 3. 并发优化
- 输入输出分离处理
- 多工作者并发转发
- 无锁数据结构

## 🔒 安全考虑

### 1. 权限控制
- 需要 root 权限进行网络抓包
- 文件访问权限检查
- 网络连接安全验证

### 2. 数据安全
- 敏感数据过滤
- 传输加密支持 (计划中)
- 访问日志记录

## 🔮 扩展性设计

### 1. 插件扩展
- 标准化的插件接口
- 反射机制动态加载
- 配置驱动的插件选择

### 2. 协议扩展
- 可插拔的编码器
- 自定义消息格式支持
- 多协议版本兼容

### 3. 存储扩展
- 多种存储后端支持
- 可配置的序列化格式
- 压缩和加密选项

## 📊 监控和诊断

### 1. 日志系统
- 分级日志记录
- 结构化日志格式
- 性能指标记录

### 2. 指标收集
- 处理速率统计
- 错误率监控
- 资源使用情况

### 3. 调试支持
- 详细的错误信息
- 调试模式输出
- 性能分析工具集成

## 🔄 部署架构

### 1. 单机部署
```
Client ←→ gRPC Server
    ↑
grpcreplay (监听端口)
    ↓
输出目标 (文件/控制台/其他服务)
```

### 2. 分布式部署
```
Client ←→ gRPC Server
    ↑
grpcreplay (捕获)
    ↓
RocketMQ
    ↓
grpcreplay (重放) → 测试环境
```

## 📈 未来规划

### 1. 功能增强
- [ ] TLS/SSL 支持
- [ ] Streaming RPC 支持
- [ ] Kafka 集成
- [ ] Web 管理界面

### 2. 性能优化
- [ ] eBPF 集成
- [ ] 更高效的数据结构
- [ ] 异步 I/O 优化

### 3. 易用性改进
- [ ] 配置文件支持
- [ ] 图形化界面
- [ ] 更多示例和文档

---

这个架构设计确保了 grpcreplay 的高性能、可扩展性和易维护性，为 gRPC 流量的捕获、分析和重放提供了完整的解决方案。
