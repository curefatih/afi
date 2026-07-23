package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentCard(t *testing.T) {
	s := &echoServer{publicURL: "http://127.0.0.1:8091/"}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	rr := httptest.NewRecorder()
	s.handleAgentCard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var card map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &card); err != nil {
		t.Fatal(err)
	}
	if card["url"] != "http://127.0.0.1:8091/" {
		t.Fatalf("url=%v", card["url"])
	}
	if card["name"] != "AFI Echo" {
		t.Fatalf("name=%v", card["name"])
	}
}

func TestMessageSendEcho(t *testing.T) {
	s := &echoServer{publicURL: "http://127.0.0.1:8091/"}
	body := `{"jsonrpc":"2.0","id":7,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"hello"}]}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleJSONRPC(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		ID     any `json:"id"`
		Result struct {
			Kind  string `json:"kind"`
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("error=%v", resp.Error)
	}
	if resp.ID != float64(7) && resp.ID != 7 {
		// json.Decoder numbers are float64
		idNum, _ := resp.ID.(float64)
		if idNum != 7 {
			t.Fatalf("id=%v", resp.ID)
		}
	}
	if resp.Result.Kind != "message" || resp.Result.Role != "agent" {
		t.Fatalf("result=%+v", resp.Result)
	}
	if len(resp.Result.Parts) != 1 || resp.Result.Parts[0].Text != "echo: hello" {
		t.Fatalf("parts=%+v", resp.Result.Parts)
	}
}

func TestAPIKeyRequired(t *testing.T) {
	s := &echoServer{publicURL: "http://127.0.0.1:8091/", apiKey: "secret"}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	rr := httptest.NewRecorder()
	s.handleAgentCard(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	rr2 := httptest.NewRecorder()
	s.handleAgentCard(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestUnknownMethod(t *testing.T) {
	s := &echoServer{publicURL: "http://127.0.0.1:8091/"}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tasks/get","params":{}}`,
	))
	rr := httptest.NewRecorder()
	s.handleJSONRPC(rr, req)
	body, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(body, []byte(`"code":-32601`)) {
		t.Fatalf("body=%s", body)
	}
}
