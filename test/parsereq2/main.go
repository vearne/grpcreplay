package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"log"

	//"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/proto"
	//dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	protov2 "google.golang.org/protobuf/proto"
	"reflect"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	pb "github.com/vearne/grpcreplay/example/service_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	//"google.golang.org/protobuf/types/dynamicpb"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"strings"
)

func constructPB() *descriptorpb.FileDescriptorProto {
	// make FileDescriptorProto
	pb := &descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("example.proto"),
		Package: proto.String("search_proto"),
		MessageType: []*descriptorpb.DescriptorProto{
			// define Foo message
			&descriptorpb.DescriptorProto{
				Name: proto.String("SearchRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("staffName"),
						JsonName: proto.String("staffName"),
						Number:   proto.Int32(1),
						Type:     descriptorpb.FieldDescriptorProto_Type(pref.StringKind).Enum(),
					},
					{
						Name:     proto.String("gender"),
						JsonName: proto.String("gender"),
						Number:   proto.Int32(2),
						Type:     descriptorpb.FieldDescriptorProto_Type(pref.BoolKind).Enum(),
					},
					{
						Name:     proto.String("age"),
						JsonName: proto.String("age"),
						Number:   proto.Int32(3),
						Type:     descriptorpb.FieldDescriptorProto_Type(pref.Uint32Kind).Enum(),
					},
				},
			},
		},
	}
	return pb
}

func main() {
	network := "tcp"
	target := "127.0.0.1:35001"
	svcAndMethod := "/SearchService/Search"

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

	inputReq := pb.SearchRequest{
		StaffName: "lisi",
		Age:       20,
		Gender:    true,
		Extra: &pb.ExtraInfo{
			JobTitle:   "software engineer",
			Location:   "Beijing",
			Department: "Back Office Department",
		},
	}
	b, err := protov2.Marshal(&inputReq)
	if err != nil {
		panic(err)
	}
	//proto.Marshal(&inputReq)
	fmt.Println("----1----", len(b))

	fileDesc := inputType.GetFile()
	fmt.Println(fileDesc.GetName())
	fmt.Println(fileDesc.GetDependencies())

	files := &descriptorpb.FileDescriptorSet{}
	files.File = append(files.File, fileDesc.AsFileDescriptorProto())
	for _, dependentItem := range fileDesc.GetDependencies() {
		files.File = append(files.File, dependentItem.AsFileDescriptorProto())
	}
	prFiles, err := protodesc.NewFiles(files)
	if err != nil {
		log.Fatal(err)
	}
	pfd, err := prFiles.FindDescriptorByName("SearchRequest")
	if err != nil {
		log.Fatal(err)
	}

	pfmd := pfd.(pref.MessageDescriptor)
	msg := dynamicpb.NewMessage(pfmd)
	//fmt.Println("msg.Type()", msg.Type())
	//cloneMsg := protov2.Clone(msg)
	fmt.Println("----1----", len(b))
	if err := proto.Unmarshal(b, msg); err != nil {
		panic(err)
	}

	jsonPrint(msg)
}

func jsonPrint(v protov2.Message) {
	b, err := protojson.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Println("jsonPrint:", string(b))
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
