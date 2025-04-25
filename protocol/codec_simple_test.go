package protocol

import (
	"fmt"
	"testing"
	"time"
)

func TestCodecSimple_MarshalUnmarshal(t *testing.T) {
	// Test cases
	tests := []struct {
		name             string
		msg              *Message
		wantErrMarshal   bool
		wantErrUnmarshal bool
	}{
		{
			name: "basic message without response",
			msg: &Message{
				Meta: Meta{
					Version:         1,
					UUID:            "test-uuid",
					Timestamp:       time.Now().Unix(),
					ContainResponse: false,
				},
				Method:   "/test.Method",
				Request:  &MsgItem{Body: "request data"},
				Response: nil,
			},
			wantErrMarshal:   false,
			wantErrUnmarshal: false,
		},
		{
			name: "message with response",
			msg: &Message{
				Meta: Meta{
					Version:         1,
					UUID:            "test-uuid-2",
					Timestamp:       time.Now().Unix(),
					ContainResponse: true,
				},
				Method:   "/test.Method2",
				Request:  &MsgItem{Body: "request data 2"},
				Response: &MsgItem{Body: "response data 2"},
			},
			wantErrMarshal:   false,
			wantErrUnmarshal: false,
		},
	}

	codec := CodecSimple{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Marshal
			data, err := codec.Marshal(tt.msg)
			if (err != nil) != tt.wantErrMarshal {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErrMarshal)
				return
			}
			if tt.wantErrMarshal {
				return
			}

			// Test Unmarshal
			got := &Message{}
			err = codec.Unmarshal(data, got)
			if (err != nil) != tt.wantErrUnmarshal {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErrUnmarshal)
				return
			}

			// Verify unmarshaled data matches original
			if tt.msg.Meta.Version != got.Meta.Version ||
				tt.msg.Meta.UUID != got.Meta.UUID ||
				tt.msg.Meta.Timestamp != got.Meta.Timestamp ||
				tt.msg.Meta.ContainResponse != got.Meta.ContainResponse ||
				tt.msg.Method != got.Method ||
				tt.msg.Request.Body != got.Request.Body {
				t.Errorf("Unmarshal() got = %+v, want %+v", got, tt.msg)
			}

			if tt.msg.Meta.ContainResponse {
				fmt.Println(tt.msg.Response.Body, got.Response.Body)
				if tt.msg.Response.Body != got.Response.Body {
					t.Errorf("Unmarshal() response got = %+v, want %+v", got.Response, tt.msg.Response)
				}
			}
		})
	}
}
