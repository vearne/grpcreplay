# grpcreplay
GrpcReplay is an open-source network monitoring tool which can record your grpc traffic and use it for shadowing, load testing, monitoring and detailed analysis.

## usage
```
./grpcreplay --input-raw="0.0.0.0:8080" --output-stdout
```

## input和output传递的消息
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