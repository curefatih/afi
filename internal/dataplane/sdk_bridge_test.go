package dataplane

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

type stubSDK struct{}

func (stubSDK) Type() string { return "stub_sdk" }
func (stubSDK) Capabilities() sdkprovider.Capabilities {
	return sdkprovider.Capabilities{Chat: true}
}
func (stubSDK) Chat(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error) {
	_ = ctx
	_ = cfg
	_ = targetModel
	_ = stream
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
		Header:     make(http.Header),
	}, nil
}

func TestRegisterSDK(t *testing.T) {
	t.Parallel()
	reg := NewRegistry().RegisterSDK(stubSDK{})
	adapter, ok := reg.Get("stub_sdk")
	if !ok {
		t.Fatal("missing")
	}
	resp, err := adapter.Chat(context.Background(), snapshot.Provider{Type: "stub_sdk"}, "m", []byte(`{}`), false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("%d", resp.StatusCode)
	}
}
