# grpcreplay
GrpcReplay is an open-source network monitoring tool which can record your grpc traffic and use it for shadowing, load testing, monitoring and detailed analysis.

## 特性
* 支持过滤器
* 可以解析Protobuf,需要grpc反射,参考[GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
* 支持多种input和output
* 支持多种gRPC请求的编码形式(可以方便的扩展)
* 支持gRPC请求重放


## 注意（请务必阅读一下）
1. 暂时只支持h2c, 不支持h2
2. 目前gRPC的编码只支持Protobuf且不支持使用Compressor。 
参考[encoding](https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md)
3. gPRC 服务端需要开启发射机制
4. 只支持Unary RCP，不支持Streaming RPC
5. 需要macOS上需要sudo
```
sudo -s
```

## 用法
```
./grpcreplay --input-raw="0.0.0.0:8080" --output-stdout
```
```
go run main.go --input-raw="0.0.0.0:8080" --output-stdout
```
```
go run main.go --input-raw="127.0.0.1:8080" --output-stdout
```

## 调试
设置日志级别
可选值: debug | info | warn | error
```
export SIMPLE_LOG_LEVEL=debug
```
