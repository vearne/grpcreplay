# grpcreplay
[![golang-ci](https://github.com/vearne/grpcreplay/actions/workflows/golang-ci.yml/badge.svg)](https://github.com/vearne/grpcreplay/actions/workflows/golang-ci.yml)

GrpcReplay 是一个网络监控工具，可以记录您的 grpc流量(Unary RPC)，并将其用于灰度测试、压测或者流量分析。


* [English README](https://github.com/vearne/grpcreplay/blob/main/README.md)

## 特性
* 支持过滤器
* 可以解析Protobuf,需要grpc反射,参考[GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
* 支持多种input和output
* 支持多种gRPC请求的编码形式(可以方便的扩展)
* 支持gRPC请求重放

## 编译
### 安装libpcap
Ubuntu
```
apt-get install -y libpcap-dev
```
Centos
```
yum install -y libpcap-devel
```
Mac
```
brew install libpcap
```
### 编译
```
make build
```

## 原理
1. 由于gRPC使用的Hpack来压缩头部，为了解决这个问题，使用了类似于tcpkill的机制，杀死旧连接，迫使client端发起新连接
2. 使用gRPC的反射机制来获取Message的定义，以便能够解析gRPC请求

## 架构图
![architecture](https://github.com/vearne/grpcreplay/raw/main/img/grpc.svg)

## 注意（请务必阅读一下）
1. 暂时只支持h2c, 不支持h2
2. 目前gRPC的编码只支持Protobuf。 
参考[encoding](https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md)
3. gPRC服务端需要开启反射 [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
4. 只支持Unary RPC，不支持Streaming RPC
5. macOS上需要sudo
```
sudo -s
```

## 用法
捕获"0.0.0.0:35001"上的gRPC请求，并打印在控制台中
```
./grpcr --input-raw="0.0.0.0:35001" --output-stdout
```
捕获"127.0.0.1:35001"上的gRPC请求，并打印在控制台中
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout
```
捕获"127.0.0.1:35001"上的gRPC请求，发往"127.0.0.1:35002"， 同时打印在控制台中
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```

指定codec   可选值: "simple" |  "json"
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --codec="simple"
```

捕获"127.0.0.1:35001"上的gRPC请求，并记录在某个文件夹中, 文件按照最大500MB的限制进行切分，并有压缩。 
注意: 目录必须已经存在且可以执行写入操作
```
./grpcr --input-raw="127.0.0.1:35001" --output-file-directory="/tmp/mycapture" --output-file-max-size=500
```
从某个文件夹中读取gRPC请求，进行重放。gRPC请求会被发往"127.0.0.1:35002"，同时打印在控制台中
```
./grpcr --input-file-directory="/tmp/mycapture" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```

捕获"127.0.0.1:35001"上的gRPC请求，只保留method后缀为Time的请求，并打印在控制台中
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --include-filter-method-match=".*Time$"
```

### 捕获的请求形如
```
{
	"headers": {
		":authority": "localhost:35001",
		":method": "POST",
		":path": "/SearchService/Search",
		":scheme": "http",
		"content-type": "application/grpc",
		"te": "trailers",
		"testkey1": "testvalue1",
		"testkey2": "testvalue2",
		"user-agent": "grpc-go/1.48.0"
	},
	"method": "/SearchService/Search",
	"request": "{\"staffName\":\"zhangsan\",\"gender\":true,\"age\":405084}"
}
```


## 调试
设置日志级别
可选值: debug | info | warn | error
```
export SIMPLE_LOG_LEVEL=debug
```

## 依赖
本项目使用了[google/gopacket](https://github.com/google/gopacket)，因而依赖`libpcap`

### 感谢
受到 [fullstorydev/grpcurl](https://github.com/fullstorydev/grpcurl) 
和 [buger/goreplay](https://github.com/buger/goreplay)的启发

## TODO
* [x] 1)发现与目标端口关联的所有连接
* [x] 2)使用旁路阻断逐个结束这些连接
* [x] 3)抓取目标端口上的请求并解析
* [x] 4)GRPC请求重放
* [x] 5)支持将GRPC请求写入控制台
* [x] 6)支持将GRPC请求写入文件
* [ ] 7)支持将GRPC请求写入kafka
* [ ] 8)支持将GRPC请求写入RocketMQ
* [x] 9)支持从文件中读取GRPC请求
* [ ] 10)支持从kafka中读取GRPC请求
* [ ] 11)支持从RocketMQ中读取GRPC请求
* [x] 12)支持自定义filter
* [ ] 13)支持TLS
