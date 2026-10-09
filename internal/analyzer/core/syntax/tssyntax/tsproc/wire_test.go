package tsproc

import (
	"encoding/json"
	"testing"
)

func TestProtocolIsPinned(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(hello{Protocol: protocol})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) != `{"Protocol":2}` {
		t.Fatalf("greeting %s", body)
	}
}
