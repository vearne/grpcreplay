# grpcreplay
GrpcReplay is a network monitoring tool that can record your grpc traffic (Unary RPC) 
and use it for grayscale testing, stress testing or traffic analysis.

## Feature
* support filter
* support to parse Protobuf, requires grpc reflection [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
* Supports various input and output plugins
* Supports multiple encoding forms of gRPC requests (can be easily extended)
* Support gRPC request replay

## Principle
1. Since gRPC uses Hpack to compress the header, in order to solve this problem, a mechanism similar to tcpkill 
is used to kill the old connection and force the client to initiate a new connection.
2. Use gRPC's reflection mechanism to get the definition of Message so that gRPC requests can be parsed.

## Architecture
![architecture](https://github.com/vearne/grpcreplay/raw/main/img/grpc.svg)

## Notice
1. Temporarily only supports h2c, not h2
2. The current gRPC encoding only supports Protobuf and does not support the use of Compressor.
   refer to [encoding](https://github.com/grpc/grpc-go/blob/master/Documentation/encoding.md)
3. The gRPC server needs to enable reflection [GRPC Server Reflection Protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#grpc-server-reflection-protocol)
4. Only supports Unary RPC, not Streaming RPC
5. Root permissions required on macOS
```
sudo -s
```

## Usage
```
./grpcr --input-raw="0.0.0.0:35001" --output-stdout
```
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout
```
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
```

Set the value of codec, optional value: "simple" |  "json"
```
./grpcr --input-raw="127.0.0.1:35001" --output-stdout --codec="simple"
```
The directory must already exist and be available for writing
```
./grpcr --input-raw="127.0.0.1:35001" --output-file-directory="/tmp/mycapture" --output-file-max-size=500
```
```
./grpcr --input-file-directory="/tmp/mycapture" --output-stdout --output-grpc="grpc://127.0.0.1:35002"
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
* [ ] 8)Support for writing GRPC requests to RocketMQ
* [x] 9)Support for reading GRPC requests from files
* [ ] 10)Support reading GRPC requests from kafka
* [ ] 11)Support for reading GRPC requests from RocketMQ
* [ ] 12)Support custom filter
* [ ] 13)support TLS