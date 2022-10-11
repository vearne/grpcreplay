# grpcreplay
GrpcReplay is an open-source network monitoring tool which can record your grpc traffic and use it for shadowing, load testing, monitoring and detailed analysis.

## 注意
需要macOS上需要sudo
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

## input和output传递的消息
```
{version} {uuid} {start-timestamp} 
{data}    // json字符串
```

data形如
```
{
	"headers": {
		"key1": [],
		"key2": []
	},
	"method": "proto.SearchService/xxx",
	"request": ""
}
```