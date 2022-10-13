package biz

import "testing"

func TestExtractAddr(t *testing.T) {
	cases := []struct {
		serverAddr, expected string
	}{
		{"grpc://192.168.1.100:8080", "192.168.1.100:8080"},
		{"grpc://192.168.1.100:8080/abc", "192.168.1.100:8080"},
		{"192.168.1.100:8080", "192.168.1.100:8080"},
		{"192.168.1.100:8080/abc", "192.168.1.100:8080"},
	}

	for _, c := range cases {
		ans, err := extractAddr(c.serverAddr)
		if err != nil {
			t.Fatalf("expectd:%v, got:%v, error:%v",
				c.expected, ans, err)
		}
		if ans != c.expected {
			t.Fatalf("expectd:%v, got:%v",
				c.expected, ans)
		}

	}
}
