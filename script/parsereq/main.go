package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/jsonpb"
	//"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	//protov2 "google.golang.org/protobuf/proto"
	"reflect"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"strings"
)

func main() {
	network := "tcp"
	target := "127.0.0.1:8080"
	svcAndMethod := "/proto.SearchService/Search"

	ctx := context.Background()

	var cc *grpc.ClientConn
	cc, err := grpcurl.BlockingDial(ctx, network, target, nil)
	if err != nil {
		panic(err)
	}
	md := grpcurl.MetadataFromHeaders(nil)
	refCtx := metadata.NewOutgoingContext(ctx, md)
	refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	descSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	fmt.Println("descSource", descSource)
	svc, method := parseSymbol(svcAndMethod)
	dsc, err := descSource.FindSymbol(svc)
	if err != nil {
		panic(err)
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		panic(err)
	}
	mtd := sd.FindMethodByName(method)
	inputType := mtd.GetInputType()
	//v := proto.MessageV2(inputType.AsProto())
	fmt.Println("mtd.GetInputType()", inputType.GetName(), inputType.GetFullyQualifiedName())
	//for _, field := range inputType.GetFields() {
	//	fmt.Println(field.GetName(), "type:", field.GetType(), "jsonName:", field.GetJSONName())
	//}

	data, err := base64.StdEncoding.DecodeString("AAAAAAYKBGdSUEM=")
	if err != nil {
		panic(err)
	}
	data = data[5:]
	//err = protov2.Unmarshal(data[5:], message)
	//if err != nil {
	//	panic(err)
	//}
	//vv, ok := inputType.(proto.Message)
	//if !ok {
	//	fmt.Println("failed to unmarshal, message is %T, want proto.Message", v)
	//}
	vv := inputType.AsProto()
	fmt.Printf("vv:%T \n", vv)
	err = proto.Unmarshal(data, vv)
	if err != nil {
		panic(err)
	}
	v := vv.(*dpb.DescriptorProto)
	fmt.Println(v.GetField())
	fmt.Println(proto.MarshalTextString(v))
	bt, err := json.Marshal(v)
	fmt.Println("json.Marshal", string(bt), err)
}

func parseSymbol(svcAndMethod string) (string, string) {
	if svcAndMethod[0] == '/' {
		svcAndMethod = svcAndMethod[1:]
	}
	pos := strings.LastIndex(svcAndMethod, "/")
	if pos < 0 {
		pos = strings.LastIndex(svcAndMethod, ".")
		if pos < 0 {
			return "", ""
		}
	}
	return svcAndMethod[:pos], svcAndMethod[pos+1:]
}

// anyResolverWithFallback can provide a fallback value for unknown
// messages that will format itself to JSON using an "@value" field
// that has the base64-encoded data for the unknown message value.
type anyResolverWithFallback struct {
	jsonpb.AnyResolver
}

func (r anyResolverWithFallback) Resolve(typeUrl string) (proto.Message, error) {
	msg, err := r.AnyResolver.Resolve(typeUrl)
	if err == nil {
		return msg, err
	}

	// Try "default" resolution logic. This mirrors the default behavior
	// of jsonpb, which checks to see if the given message name is registered
	// in the proto package.
	fmt.Println("typeUrl", typeUrl)
	mname := typeUrl
	if slash := strings.LastIndex(mname, "/"); slash >= 0 {
		mname = mname[slash+1:]
	}
	//lint:ignore SA1019 new non-deprecated API requires other code changes; deferring...
	mt := proto.MessageType(mname)
	if mt != nil {
		return reflect.New(mt.Elem()).Interface().(proto.Message), nil
	}

	// finally, fallback to a special placeholder that can marshal itself
	// to JSON using a special "@value" property to show base64-encoded
	// data for the embedded message
	return &unknownAny{TypeUrl: typeUrl, Error: fmt.Sprintf("%s is not recognized; see @value for raw binary message data", mname)}, nil
}

type unknownAny struct {
	TypeUrl string `json:"@type"`
	Error   string `json:"@error"`
	Value   string `json:"@value"`
}

func (a *unknownAny) MarshalJSONPB(jsm *jsonpb.Marshaler) ([]byte, error) {
	if jsm.Indent != "" {
		return json.MarshalIndent(a, "", jsm.Indent)
	}
	return json.Marshal(a)
}

func (a *unknownAny) Unmarshal(b []byte) error {
	a.Value = base64.StdEncoding.EncodeToString(b)
	return nil
}

func (a *unknownAny) Reset() {
	a.Value = ""
}

func (a *unknownAny) String() string {
	b, err := a.MarshalJSONPB(&jsonpb.Marshaler{})
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err.Error())
	}
	return string(b)
}

func (a *unknownAny) ProtoMessage() {
}
