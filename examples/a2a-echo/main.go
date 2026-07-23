// Command a2a-echo is a minimal Agent2Agent (A2A) JSON-RPC upstream for local AFI testing.
// It serves an Agent Card and echoes message/send text back — no LLM calls.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", envOr("A2A_ECHO_ADDR", ":8091"), "listen address")
	publicURL := flag.String("url", envOr("A2A_ECHO_URL", ""), "public agent URL advertised in the Agent Card (default http://127.0.0.1:<port>/)")
	apiKey := flag.String("api-key", os.Getenv("A2A_ECHO_API_KEY"), "optional Bearer token required on requests")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	baseURL := strings.TrimRight(*publicURL, "/")
	if baseURL == "" {
		host := ln.Addr().String()
		if strings.HasPrefix(host, "[::]:") {
			host = "127.0.0.1:" + strings.TrimPrefix(host, "[::]:")
		} else if strings.HasPrefix(host, "0.0.0.0:") {
			host = "127.0.0.1:" + strings.TrimPrefix(host, "0.0.0.0:")
		} else if strings.HasPrefix(host, ":") {
			host = "127.0.0.1" + host
		}
		baseURL = "http://" + host
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	srv := &echoServer{publicURL: baseURL, apiKey: strings.TrimSpace(*apiKey)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/agent-card.json", srv.handleAgentCard)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	// JSON-RPC endpoint (Upstream URL should be this base, e.g. http://127.0.0.1:8091/).
	mux.HandleFunc("POST /{$}", srv.handleJSONRPC)

	log.Printf("a2a-echo listening on %s (agent url %s)", ln.Addr(), baseURL)
	if srv.apiKey != "" {
		log.Printf("Bearer auth required (A2A_ECHO_API_KEY)")
	}
	if err := http.Serve(ln, mux); err != nil {
		log.Fatal(err)
	}
}

type echoServer struct {
	publicURL string
	apiKey    string
}

func (s *echoServer) authorize(w http.ResponseWriter, r *http.Request) bool {
	if s.apiKey == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(auth[len(prefix):]) != s.apiKey {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "unauthorized", "type": "authentication_error"},
		})
		return false
	}
	return true
}

func (s *echoServer) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}
	card := map[string]any{
		"name":               "AFI Echo",
		"description":        "Local A2A echo agent for AFI gateway testing. Replies with the same text you send.",
		"url":                s.publicURL,
		"version":            "0.1.0",
		"protocolVersion":    "0.3.0",
		"preferredTransport": "JSONRPC",
		"capabilities": map[string]any{
			"streaming": false,
		},
		"defaultInputModes":  []string{"text"},
		"defaultOutputModes": []string{"text"},
		"skills": []map[string]any{
			{
				"id":          "echo",
				"name":        "Echo",
				"description": "Returns the user message text prefixed with echo:",
				"tags":        []string{"echo", "demo"},
			},
		},
	}
	writeJSON(w, http.StatusOK, card)
}

func (s *echoServer) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONRPCError(w, nil, -32700, "failed to read body")
		return
	}
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error")
		return
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, -32600, "jsonrpc must be 2.0")
		return
	}

	switch strings.TrimSpace(req.Method) {
	case "message/send":
		s.handleMessageSend(w, req.ID, req.Params)
	case "tasks/get", "tasks/cancel", "message/stream":
		writeJSONRPCError(w, req.ID, -32601, "method not implemented: "+req.Method)
	default:
		writeJSONRPCError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *echoServer) handleMessageSend(w http.ResponseWriter, id json.RawMessage, params json.RawMessage) {
	text, contextID, err := extractUserText(params)
	if err != nil {
		writeJSONRPCError(w, id, -32602, err.Error())
		return
	}
	if contextID == "" {
		contextID = newID("ctx")
	}
	echo := "echo: " + text
	if text == "" {
		echo = "echo: (empty message)"
	}
	result := map[string]any{
		"kind":      "message",
		"role":      "agent",
		"messageId": newID("msg"),
		"contextId": contextID,
		"parts": []map[string]any{
			{"kind": "text", "text": echo},
		},
		"metadata": map[string]any{
			"agent":     "afi-echo",
			"echoed_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"jsonrpc": "2.0",
		"id":      rawOrNull(id),
		"result":  result,
	})
}

func extractUserText(params json.RawMessage) (text, contextID string, err error) {
	if len(params) == 0 {
		return "", "", fmt.Errorf("params required")
	}
	var p struct {
		Message *struct {
			Role      string           `json:"role"`
			ContextID string           `json:"contextId"`
			Parts     []map[string]any `json:"parts"`
		} `json:"message"`
		ContextID string `json:"contextId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", "", fmt.Errorf("invalid params")
	}
	if p.ContextID != "" {
		contextID = p.ContextID
	}
	if p.Message == nil {
		return "", contextID, fmt.Errorf("params.message required")
	}
	if p.Message.ContextID != "" {
		contextID = p.Message.ContextID
	}
	var parts []string
	for _, part := range p.Message.Parts {
		if t, ok := part["text"].(string); ok && t != "" {
			parts = append(parts, t)
			continue
		}
		// Some clients nest text under data.
		if data, ok := part["data"].(map[string]any); ok {
			if t, ok := data["text"].(string); ok && t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, "\n"), contextID, nil
}

func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"jsonrpc": "2.0",
		"id":      rawOrNull(id),
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func rawOrNull(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(id, &v); err != nil {
		return nil
	}
	return v
}

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + "_" + hex.EncodeToString(b[:])
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
