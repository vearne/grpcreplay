## generate go file
```
protoc --go_out=. --go_opt=paths=source_relative\
 --go-grpc_out=. --go-grpc_opt=paths=source_relative\
  --go-grpc_opt=require_unimplemented_servers=false *.proto
```

## Verify reflection interface
```
grpcurl -plaintext localhost:35001 list
grpcurl -plaintext localhost:35001 list SearchService
grpcurl -plaintext localhost:35001 describe SearchService.Search
grpcurl -plaintext localhost:35001 describe .SearchRequest

grpcurl -plaintext -format json -d '{
	"staffName": "zhangsan",
	"gender": true,
	"age": 10,
	"extra": {
		"jobTitle": "software engineer",
		"location": "Beijing",
		"department": "Back Office Department"
	}
}' localhost:35001 SearchService.Search
```