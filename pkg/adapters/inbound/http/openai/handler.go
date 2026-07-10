package openai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type OpenAIHandler struct {
	gatewayUseCase ports.LLMGatewayUseCase
	authUseCase    ports.AuthUseCase
}

func NewHandler(gateway ports.LLMGatewayUseCase, auth ports.AuthUseCase) *OpenAIHandler {
	return &OpenAIHandler{
		gatewayUseCase: gateway,
		authUseCase:    auth,
	}
}

// ServeHTTP acts as our high-performance multiplexer endpoint.
func (h *OpenAIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Enforce Authentication Layer Boundary
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, `{"error": "Missing or malformed Authorization header"}`, http.StatusUnauthorized)
		return
	}
	rawKey := strings.TrimPrefix(authHeader, "Bearer ")

	// Authenticate key and unpack full platform tenant metadata context hierarchy
	reqCtx, err := h.authUseCase.AuthenticateKey(r.Context(), rawKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API Key token provided"}`))
		return
	}

	// Attach trace properties
	reqCtx.CallerIP = r.RemoteAddr
	reqCtx.ReceivedAt = time.Now()
	reqCtx.TraceID = fmt.Sprintf("trc_%d", time.Now().UnixNano())

	// 2. Unmarshal the Incoming OpenAI-compatible JSON Request
	var incomingReq OpenAIRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&incomingReq); err != nil {
		http.Error(w, `{"error": "Malformed JSON payload"}`, http.StatusBadRequest)
		return
	}

	// 3. Map to clean Internal Representation (Domain)
	internalReq := h.mapToDomainRequest(&incomingReq, reqCtx)

	// 4. Branch execution matrix based on Streaming vs Unary
	if incomingReq.Stream {
		h.handleStreamingExecution(w, r, internalReq)
	} else {
		h.handleUnaryExecution(w, r, internalReq)
	}
}

func (h *OpenAIHandler) handleUnaryExecution(w http.ResponseWriter, r *http.Request, req *domain.InternalRequest) {
	resp, err := h.gatewayUseCase.ExecuteUnary(r.Context(), req)
	if err != nil {
		h.writeErrorResponse(w, err)
		return
	}

	// Map internal domain response object schema back onto OpenAI standard output wire format
	openAIResp := mapFromDomainResponse(resp)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(openAIResp)
}

func (h *OpenAIHandler) handleStreamingExecution(w http.ResponseWriter, r *http.Request, req *domain.InternalRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported by current connection wrapper", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	chunks, errCh := h.gatewayUseCase.ExecuteStream(ctx, req)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if err != nil {
				log.Printf("[STREAM ERROR] TraceID: %s | %v", req.Metadata.TraceID, err)
			}
			return
		case chunk, ok := <-chunks:
			if !ok {
				// Send strict OpenAI stream terminator sequence
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			// Format chunk back to standard OpenAI response shape
			wireChunk := mapFromDomainChunk(&chunk)
			jsonBytes, _ := json.Marshal(wireChunk)

			fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
			flusher.Flush() // Flush instantly out over the wire
		}
	}
}

func (h *OpenAIHandler) writeErrorResponse(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"error": {"message": %q}}`, err.Error())
}
