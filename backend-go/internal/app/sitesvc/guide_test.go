package sitesvc

import (
	"context"
	"testing"
)

func TestResolveHostIPv4Literal(t *testing.T) {
	got := resolveHostIPv4(context.Background(), "127.0.0.1")
	if len(got) != 1 || got[0] != "127.0.0.1" {
		t.Fatalf("literal: %v", got)
	}
	if resolveHostIPv4(context.Background(), "  ") != nil {
		t.Fatal("blank host")
	}
}
