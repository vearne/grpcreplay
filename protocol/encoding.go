package protocol

import "strings"

type Codec interface {
	// Marshal returns the wire format of v.
	Marshal(v *Message) ([]byte, error)
	// Unmarshal parses the wire format into v.
	Unmarshal(data []byte, v *Message) error
	// Name returns the name of the Codec implementation. The returned string
	// will be used as part of content type in transmission.  The result must be
	// static; the result cannot change between calls.
	Name() string
}

var registeredCodecs = make(map[string]Codec)

func RegisterCodec(codec Codec) {
	if codec == nil {
		panic("cannot register a nil Codec")
	}
	if codec.Name() == "" {
		panic("cannot register Codec with empty string result for Name()")
	}
	contentSubtype := strings.ToLower(codec.Name())
	registeredCodecs[contentSubtype] = codec
}

// The content-subtype is expected to be lowercase.
func GetCodec(codecType string) Codec {
	return registeredCodecs[codecType]
}
