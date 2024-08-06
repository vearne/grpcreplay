package main

import (
	"fmt"
	pb "github.com/vearne/grpcreplay/example/service_proto"
	"github.com/vearne/grpcreplay/example/service_proto/another"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
)

func main() {
	inputReq := pb.SearchRequest{
		StaffName: "lisi",
		Age:       20,
		Gender:    true,
		Extra: &pb.ExtraInfo{
			JobTitle:   "software engineer",
			Location:   "Beijing",
			Department: &another.Department{Id: 2001},
		},
	}
	b, err := proto.Marshal(&inputReq)
	if err != nil {
		panic(err)
	}
	WriteFile("/tmp/inputReq.bin", b)
}

func WriteFile(filename string, data []byte) {
	// 使用 ioutil.WriteFile 写入文件
	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		fmt.Println("写入文件失败:", err)
		return
	}

	fmt.Println("数据已成功写入文件")
}
