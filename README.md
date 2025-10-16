# grpcreplay
[![golang-ci](https://github.com/vearne/grpcreplay/actions/workflows/golang-ci.yml/badge.svg)](https://github.com/vearne/grpcreplay/actions/workflows/golang-ci.yml)

GrpcReplay is a network monitoring tool that can record your grpc traffic (Unary RPC) 
and use it for grayscale testing, stress testing or traffic analysis.

* [中文 README](https://github.com/vearne/grpcreplay/blob/main/README_zh.md)
* [Architecture Design](https://github.com/vearne/grpcreplay/blob/main/ARCHITECTURE.md)
* [Usage Examples and Best Practices](https://github.com/vearne/grpcreplay/blob/main/EXAMPLES.md)
* [Contributing Guide](https://github.com/vearne/grpcreplay/blob/main/CONTRIBUTING.md)

## Feature
* support filter
* support to parse Protobuf, **requires grpc reflection** [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
* Supports various input and output plugins
* Supports multiple encoding forms of gRPC requests (can be easily extended)
* Support gRPC request replay and replay at multiple speeds

## Test
Python/Java/Golang gRPC all passed the test
### Python Demo
* [demo](https://github.com/grpc/grpc/blob/master/examples/python/helloworld/greeter_server_with_reflection.py)
* [how to use demo](https://grpc.io/docs/languages/python/quickstart/)
### Java Demo
* [demo](https://github.com/grpc/grpc-java/tree/master/examples/example-reflection)
* [how to use demo](https://github.com/grpc/grpc-java/blob/master/examples/example-reflection/README.md)
### Golang Demo
* [demo](https://github.com/vearne/grpcreplay/tree/main/example)


## Compile
### install libpcap
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
### compile
```
make build
```

## Principle
1. Since gRPC uses Hpack to compress the header, in order to solve this problem, a mechanism similar to tcpkill 
is used to kill the old connection and force the client to initiate a new connection.
2. Use gRPC's reflection mechanism to get the definition of Message so that gRPC requests can be parsed.

## Architecture
![architecture](https://github.com/vearne/grpcreplay/raw/main/img/grpc.svg)

## Notice
1. Temporarily only supports h2c, not h2
2. The current gRPC encoding only supports Protobuf.
   refer to [encoding](https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md)
3. Parsing Protobuf requires providing protobuf definition, which supports the following two methods.<br/>
3.1 gRPC server enables reflection [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)  (默认)<br/>
3.2 provide local protobuf definition file
```
./grpcr --input-raw="0.0.0.0:35001" --output-stdout --record-response --proto=./proto
```
`--proto` You can specify a file or folder. If it is a folder, all files with the suffix ".proto" will be loaded.
4.  Only supports Unary RPC, not Streaming RPC
5. Root permissions required on macOS
```
sudo -s
```
6. client and server must be on different hosts

## Usage
Capture gRPC request on "0.0.0.0:35001" and print in console
```
./grpcr --input-raw="0.0.0.0:35001" --output-stdout --record-response
```
`--record-response`(optional): record response

Capture gRPC request on "127.0.0.1:35001" and print in console
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --record-response
```
Capture the gRPC request on "127.0.0.1:35001", send it to "127.0.0.1:35002", and print it in the console
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```

Set the value of codec, optional value: "simple" |  "json"
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --codec="simple"
```
Capture the gRPC request on "127.0.0.1:35001" and record it in a folder. 
The file is divided according to the maximum limit of 500MB and compressed.
Note: The directory must already exist and be writeable.
```
./grpcr --input-raw="127.0.0.1:35001" --output-file-directory="/tmp/mycapture" --output-file-max-size=500
```
Read gRPC requests from a folder and replay them. gRPC requests will be sent to "127.0.0.1:35002" 
and printed in the console
```
./grpcr --input-file-directory="/tmp/mycapture" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```
Hint: You can use `input-file-replay-speed` to speed up the replay
```
--input-file-replay-speed=10
```

Capture gRPC requests on "127.0.0.1:35001", 
keep only requests whose method suffix is Time, and print them in the console
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --include-filter-method-match=".*Time$"
```

Capture the gRPC request on "127.0.0.1:35001" and send it to the test topic of RocketMQ
```
./grpcr --input-raw="127.0.0.1:35001" --output-rocketmq-name-server="192.168.2.100:9876" --output-rocketmq-topic="test"
```

Get the gRPC request from RocketMQ, send it to "127.0.0.1:35001", and print it in the console
```
./grpcr --input-rocketmq-name-server="192.168.2.100:9876" --input-rocketmq-topic="test" --output-stdout --output-grpc="grpc://127.0.0.1 :35001"
```


### the captured data
#### --codec="simple"
```
2 f8762dc4-20fa-11f0-a55f-5626e1cdcfe2 1745492273089274000 1
/SearchService/CurrentTime
{"headers":{":authority":"10.2.139.146:35001",":method":"POST",":path":"/SearchService/CurrentTime",":scheme":"http","content-type":"application/grpc","grpc-accept-encoding":"gzip","te":"trailers","testkey3":"testvalue3","testkey4":"testvalue4","user-agent":"grpc-go/1.65.0"},"body":"{\"requestId\":\"2\"}"}
{"headers":{":status":"200","content-type":"application/grpc","grpc-message":"","grpc-status":"0"},"body":"{\"currentTime\":\"2025-04-24T18:57:49+08:00\"}"}
```
#### --codec="json"
```
{
	"meta": {
		"version": 2,
		"uuid": "644e70a0-20fc-11f0-9ba0-5626e1cdcfe2",
		"timestamp": 1745492883519504000,
		"containResponse": true
	},
	"method": "/SearchService/CurrentTime",
	"request": {
		"headers": {
			":authority": "10.2.139.146:35001",
			":method": "POST",
			":path": "/SearchService/CurrentTime",
			":scheme": "http",
			"content-type": "application/grpc",
			"grpc-accept-encoding": "gzip",
			"te": "trailers",
			"testkey3": "testvalue3",
			"testkey4": "testvalue4",
			"user-agent": "grpc-go/1.65.0"
		},
		"body": "{\"requestId\":\"2\"}"
	},
	"response": {
		"headers": {
			":status": "200",
			"content-type": "application/grpc",
			"grpc-message": "",
			"grpc-status": "0"
		},
		"body": "{\"currentTime\":\"2025-04-24T19:08:02+08:00\"}"
	}
}
```

## Debug
Set the log level
Optional value: debug | info | warn | error
```
export SIMPLE_LOG_LEVEL=debug
```

## Dependency
This project uses [google/gopacket](https://github.com/google/gopacket) and therefore depends on `libpcap`

### Thanks
inspired by [fullstorydev/grpcurl](https://github.com/fullstorydev/grpcurl) 
and [buger/goreplay](https://github.com/buger/goreplay)

## TODO
* [x] 1)Discover all connections associated with the target port
* [x] 2)Use bypass blocking to end these connections one by one
* [x] 3)Grab the request on the target port and parse it
* [x] 4)GRPC request replay
* [x] 5)Supports writing GRPC requests to the console
* [x] 6)Support for writing GRPC requests to files
* [ ] 7)Support for writing GRPC requests to kafka
* [x] 8)Support for writing GRPC requests to RocketMQ
* [x] 9)Support for reading GRPC requests from files
* [ ] 10)Support reading GRPC requests from kafka
* [x] 11)Support for reading GRPC requests from RocketMQ
* [x] 12)Support custom filter
* [ ] 13)support TLS
* [x] 14)Optimize the processing speed of output_grpc

## donate
![donate](https://github.com/vearne/grpcreplay/raw/main/img/donate.jpg)