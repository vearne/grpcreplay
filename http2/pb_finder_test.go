package http2

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestFilePBFinder(t *testing.T) {
	files := []string{"./testdata/common.proto", "./testdata/search.proto",
		"./testdata/another/department.proto"}
	finder := NewFilePBFinder(files)
	protoMsg, err := finder.Get("/SearchService/Search")
	assert.Nil(t, err, "get /SearchService/Search")
	str, jsonErr := toJsonStr(protoMsg.InType)
	t.Logf("in:%v, err:%v", str, jsonErr)

	protoMsg2, err2 := finder.Get("/SearchService/SendMuchData")
	assert.Nil(t, err2, "get /SearchService/SendMuchData")
	str, jsonErr = toJsonStr(protoMsg2.InType)
	t.Logf("in:%v, err:%v", str, jsonErr)

}

func toJsonStr(pbMsg proto.Message) (string, error) {
	result, err := protojson.Marshal(pbMsg)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
