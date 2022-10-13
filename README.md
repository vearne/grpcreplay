# grpcreplay
GrpcReplay 是一个网络监控工具，可以记录您的 grpc流量(Unary RPC)，并将其用于灰度测试、压测或者流量分析。

## 特性
* 支持过滤器
* 可以解析Protobuf,需要grpc反射,参考[GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
* 支持多种input和output
* 支持多种gRPC请求的编码形式(可以方便的扩展)
* 支持gRPC请求重放

## 原理
1. 由于gRPC使用的Hpack来压缩头部，为了解决这个问题，使用了类似于tcpkill的机制，杀死旧连接，迫使client端发起新连接
2. 使用gRPC的反射机制来获取Message的定义，以便能够解析gRPC请求

## 架构图
![architecture](https://github.com/vearne/grpcreplay/raw/main/img/grpc.svg)

## 注意（请务必阅读一下）
1. 暂时只支持h2c, 不支持h2
2. 目前gRPC的编码只支持Protobuf且不支持使用Compressor。 
参考[encoding](https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md)
3. gPRC服务端需要开启反射 [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
4. 只支持Unary RPC，不支持Streaming RPC
5. 需要macOS上需要sudo
```
sudo -s
```

## 用法
```
./grpcr --input-raw="0.0.0.0:35001" --output-stdout
```
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout
```
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```
指定codec   可选值: "simple" |  "json"
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --codec="simple"

## 调试
设置日志级别
可选值: debug | info | warn | error
```
export SIMPLE_LOG_LEVEL=debug
```
## TODO
* [x] 1)发现与目标端口关联的所有连接
* [x] 2)使用旁路阻断逐个结束这些连接
* [x] 3)抓取目标端口上的请求并解析
* [x] 4)GRPC请求重放
* [x] 5)支持将GRPC请求写入控制台
* [ ] 6)支持将GRPC请求写入文件
* [ ] 7)支持将GRPC请求写入kafka
* [ ] 8)支持将GRPC请求写入RocketMQ
* [ ] 9)支持自定义filter
* [ ] 10)支持TLS
