// Command grpcecho is a sample gRPC extension plugin (ChatIR provider + BeforeCall/BeforeChat).
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
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/internal/adapters/grpcprovider"
	"github.com/curefatih/afi/sdk/chatir"
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
			extensionv1.Capability_CAPABILITY_HOOK_BEFORE_CHAT,
		},
	}, nil
}

func (s *server) ChatIR(ctx context.Context, req *extensionv1.ChatIRRequest) (*extensionv1.ChatIRResponse, error) {
	_ = ctx
	irReq := grpcprovider.ChatIRRequestFromProto(req.GetRequest())
	if req.GetRequest() != nil && req.GetRequest().GetStream() {
		return nil, fmt.Errorf("use ChatIRStream for streaming requests")
	}
	model := req.GetTargetModel()
	if model == "" {
		model = irReq.Model
	}
	userText := lastUserText(irReq)
	content := "grpcecho: " + userText
	return grpcprovider.ChatIRResponseProto(chatirResult(model, userText, content)), nil
}

func (s *server) ChatIRStream(req *extensionv1.ChatIRRequest, stream extensionv1.Provider_ChatIRStreamServer) error {
	irReq := grpcprovider.ChatIRRequestFromProto(req.GetRequest())
	model := req.GetTargetModel()
	if model == "" {
		model = irReq.Model
	}
	userText := lastUserText(irReq)
	content := "grpcecho: " + userText
	events := []chatir.StreamEvent{
		{Kind: chatir.StreamMessageStart, ID: "chatcmpl-grpcecho", Model: model, Role: "assistant"},
		{Kind: chatir.StreamTextDelta, Text: content},
		{
			Kind: chatir.StreamMessageEnd, FinishReason: "stop",
			Usage: &chatir.Usage{
				PromptTokens:     int64(max(1, len(strings.Fields(userText)))),
				CompletionTokens: int64(max(1, len(strings.Fields(content)))),
			},
		},
	}
	for _, ev := range events {
		if err := stream.Send(grpcprovider.ChatIRStreamEventProto(ev)); err != nil {
			return err
		}
	}
	return nil
}

func lastUserText(req chatir.Request) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			if req.Messages[i].Content != "" {
				return req.Messages[i].Content
			}
			for _, part := range req.Messages[i].Parts {
				if part.Type == chatir.ContentText && part.Text != "" {
					return part.Text
				}
			}
		}
	}
	return ""
}

func chatirResult(model, userText, content string) chatir.Result {
	return chatir.Result{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"application/json"}},
		Response: &chatir.Response{
			ID: "chatcmpl-grpcecho", Model: model, Role: "assistant", Content: content,
			FinishReason: "stop",
			Usage: chatir.Usage{
				PromptTokens:     int64(max(1, len(strings.Fields(userText)))),
				CompletionTokens: int64(max(1, len(strings.Fields(content)))),
			},
		},
	}
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
	typed := grpcprovider.ChatIRRequestFromProto(req.GetRequest())
	return &extensionv1.BeforeChatResponse{Request: grpcprovider.ChatIRRequestProto(typed)}, nil
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
