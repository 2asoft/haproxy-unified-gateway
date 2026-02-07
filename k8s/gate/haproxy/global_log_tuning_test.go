package haproxy

import (
	"testing"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
)

func TestParseGlobalLogTuningConfigUsesGlobalScope(t *testing.T) {
	logLineLength := int64(4096)
	httpLogURILen := int64(8192)

	parsed, err := parseGlobalLogTuningConfig(v3.ControllerConfSpec{
		HaproxyGlobal: &v3.HaproxyGlobal{
			LogLineLength: &logLineLength,
			HTTPLogURILen: &httpLogURILen,
		},
	})
	if err != nil {
		t.Fatalf("parseGlobalLogTuningConfig returned error: %v", err)
	}
	if parsed.LogLineLength == nil || int64(*parsed.LogLineLength) != logLineLength {
		t.Fatalf("expected LogLineLength=%d, got %#v", logLineLength, parsed.LogLineLength)
	}
	if parsed.HTTPLogURILen == nil || int64(*parsed.HTTPLogURILen) != httpLogURILen {
		t.Fatalf("expected HTTPLogURILen=%d, got %#v", httpLogURILen, parsed.HTTPLogURILen)
	}
}

func TestParseGlobalLogTuningConfigReturnsEmptyWhenGlobalScopeIsUnset(t *testing.T) {
	parsed, err := parseGlobalLogTuningConfig(v3.ControllerConfSpec{
		HaproxyDefaults: &v3.HaproxyDefaults{
			LogFormat: "'%ci'",
		},
	})
	if err != nil {
		t.Fatalf("parseGlobalLogTuningConfig returned error: %v", err)
	}
	if parsed.LogLineLength != nil {
		t.Fatalf("expected LogLineLength=nil, got %#v", parsed.LogLineLength)
	}
	if parsed.HTTPLogURILen != nil {
		t.Fatalf("expected HTTPLogURILen=nil, got %#v", parsed.HTTPLogURILen)
	}
}

func TestParseGlobalLogTuningConfigRejectsInvalidValues(t *testing.T) {
	invalidValue := int64(0)

	_, err := parseGlobalLogTuningConfig(v3.ControllerConfSpec{
		HaproxyGlobal: &v3.HaproxyGlobal{
			LogLineLength: &invalidValue,
		},
	})
	if err == nil {
		t.Fatal("expected error for non-positive log line length")
	}
}
