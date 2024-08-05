package http2

const (
	CodecProtobuf = iota
	CodecOther
)

const (
	HeaderSize            = 9
	LengthSize            = 3
	ConnectionPrefaceSize = 24
	StreamArraySize       = 10000
	SettingFormatItemSize = 48
)

const (
	DirUnknown = iota
	DirIncoming
	DirOutcoming
)

var DirStr map[Dir]string

func init() {
	DirStr = make(map[Dir]string)
	DirStr[DirUnknown] = "DirUnknown"
	DirStr[DirIncoming] = "DirIncoming"
	DirStr[DirOutcoming] = "DirOutcoming"
}

func GetDirection(d Dir) string {
	name, ok := DirStr[d]
	if ok {
		return name
	}
	return "UNKNOW"
}

// The format of the payload: compressed or not?
type payloadFormat uint8

const (
	// nolint: deadcode,varcheck
	compressionNone payloadFormat = 0 // no compression
	compressionMade payloadFormat = 1 // compressed
)
