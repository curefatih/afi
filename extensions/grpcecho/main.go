// Command grpcecho is a sample gRPC extension plugin (ChatProvider + BeforeCall).
//
// Listen address comes from AFI_PLUGIN_SOCK (set by the gateway when spawning)
// or -addr. Example gateway config:
//
//	gateway:
//	  grpc_extensions:
//	    - id: grpcecho
//	      command: ["./bin/grpcecho"]
//
// Create a control-plane provider with type "grpcecho" and a route to use it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/internal/adapters/grpcprovider"
	"google.golang.org/grpc"
)

const providerType = "grpcecho"

func main() {
	addrFlag := flag.String("addr", "", "listen address (unix path or host:port); defaults to AFI_PLUGIN_SOCK")
	flag.Parse()

	addr := strings.TrimSpace(*addrFlag)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv(grpcprovider.EnvPluginSock))
	}
	if addr == "" {
		log.Fatalf("set -addr or %s", grpcprovider.EnvPluginSock)
	}

	lis, err := listen(addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	defer lis.Close()
	log.Printf("grpcecho listening on %s", addr)

	srv := grpc.NewServer()
	plugin := &server{}
	extensionv1.RegisterExtensionServer(srv, plugin)
	extensionv1.RegisterProviderServer(srv, plugin)
	extensionv1.RegisterHookServer(srv, plugin)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func listen(addr string) (net.Listener, error) {
	addr = strings.TrimSpace(addr)
	switch {
	case strings.HasPrefix(addr, "unix://"):
		path := strings.TrimPrefix(addr, "unix://")
		_ = os.Remove(path)
		return net.Listen("unix", path)
	case strings.HasPrefix(addr, "unix:"):
		path := strings.TrimPrefix(addr, "unix:")
		_ = os.Remove(path)
		return net.Listen("unix", path)
	case strings.HasPrefix(addr, "/") || (!strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[")):
		// Absolute path or bare filename → unix socket (gateway passes a path).
		_ = os.Remove(addr)
		return net.Listen("unix", addr)
	default:
		return net.Listen("tcp", addr)
	}
}

type server struct {
	extensionv1.UnimplementedExtensionServer
	extensionv1.UnimplementedProviderServer
	extensionv1.UnimplementedHookServer
}

func (s *server) Handshake(ctx context.Context, req *extensionv1.HandshakeRequest) (*extensionv1.HandshakeResponse, error) {
	_ = ctx
	_ = req
	return &extensionv1.HandshakeResponse{
		Name:         "grpcecho",
		Version:      "1.0.0",
		ProviderType: providerType,
		Capabilities: []extensionv1.Capability{
			extensionv1.Capability_CAPABILITY_PROVIDER_CHAT,
			extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CALL,
		},
	}, nil
}

func (s *server) Chat(ctx context.Context, req *extensionv1.ChatRequest) (*extensionv1.ChatResponse, error) {
	_ = ctx
	if req.GetStream() {
		return nil, fmt.Errorf("streaming is not supported for provider type %q", providerType)
	}
	body := req.GetBody()
	model := req.GetTargetModel()
	var chatReq struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &chatReq); err != nil {
		return nil, fmt.Errorf("invalid chat body: %w", err)
	}
	if model == "" {
		model = chatReq.Model
	}
	userText := ""
	for i := len(chatReq.Messages) - 1; i >= 0; i-- {
		if chatReq.Messages[i].Role == "user" {
			userText = chatReq.Messages[i].Content
			break
		}
	}
	content := "grpcecho: " + userText
	payload := map[string]any{
		"id":      "chatcmpl-grpcecho",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]string{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]int{
			"prompt_tokens":     max(1, len(strings.Fields(userText))),
			"completion_tokens": max(1, len(strings.Fields(content))),
			"total_tokens":      0,
		},
	}
	u := payload["usage"].(map[string]int)
	u["total_tokens"] = u["prompt_tokens"] + u["completion_tokens"]
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &extensionv1.ChatResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       raw,
	}, nil
}

func (s *server) BeforeCall(ctx context.Context, req *extensionv1.BeforeCallRequest) (*extensionv1.BeforeCallResponse, error) {
	_ = ctx
	call := req.GetCall()
	if call == nil {
		return &extensionv1.BeforeCallResponse{
			Decision: &extensionv1.CallDecision{Allow: true},
		}, nil
	}
	if call.RequestHeaders == nil {
		call.RequestHeaders = map[string]string{}
	}
	call.RequestHeaders["X-Afi-Grpc-Echo"] = "1"
	return &extensionv1.BeforeCallResponse{
		Decision: &extensionv1.CallDecision{Allow: true},
		Call:     call,
	}, nil
}

func (s *server) AfterCall(ctx context.Context, req *extensionv1.AfterCallRequest) (*extensionv1.AfterCallResponse, error) {
	_ = ctx
	_ = req
	return &extensionv1.AfterCallResponse{}, nil
}

func (s *server) BeforeChat(ctx context.Context, req *extensionv1.BeforeChatRequest) (*extensionv1.BeforeChatResponse, error) {
	_ = ctx
	body := req.GetBody()
	if body == nil {
		body = []byte{}
	}
	return &extensionv1.BeforeChatResponse{Body: append([]byte(nil), body...)}, nil
}

func (s *server) AfterChat(ctx context.Context, req *extensionv1.AfterChatRequest) (*extensionv1.AfterChatResponse, error) {
	_ = ctx
	_ = req
	return &extensionv1.AfterChatResponse{}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
