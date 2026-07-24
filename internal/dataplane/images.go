package dataplane

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/objectstore"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/modelcatalog"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func modelLooksLikeImage(requested, target string) bool {
	for _, m := range []string{requested, target} {
		s := strings.ToLower(strings.TrimSpace(m))
		if strings.Contains(s, "dall-e") || strings.Contains(s, "gpt-image") ||
			strings.Contains(s, "imagen") || strings.Contains(s, "image") {
			return true
		}
	}
	if entry, ok := modelcatalog.Lookup("openai", target); ok && entry.IsImage() {
		return true
	}
	if entry, ok := modelcatalog.Lookup("openai", requested); ok && entry.IsImage() {
		return true
	}
	return false
}

func imagesOpenAICompatible(typ string) bool {
	return typ == "openai" || typ == "openai_compatible"
}

func (p *Pipeline) handleImagesGenerations(w http.ResponseWriter, r *http.Request) {
	reqID := kernel.NewRequestID()
	ctx := kernel.WithRequestID(r.Context(), reqID)
	log := p.Log.With("request_id", reqID)
	start := time.Now()

	rawKey, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "missing or invalid authorization", "type": "invalid_request_error"},
		})
		return
	}
	snap := p.Holder.Get()
	if snap == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"message": "no snapshot loaded", "type": "server_error"},
		})
		return
	}
	key, ok := snap.LookupKey(rawKey)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "invalid api key", "type": "invalid_request_error"},
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "failed to read body", "type": "invalid_request_error"},
		})
		return
	}
	var reqBody struct {
		Model          string `json:"model"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil || reqBody.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "model is required", "type": "invalid_request_error"},
		})
		return
	}

	call := newCallContext(key, reqBody.Model, "/v1/images/generations", ModalityImage, false, body, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	body = call.Body

	route, provider, ok := snap.LookupRoute(key.OrganizationID, reqBody.Model)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no route for model", "type": "invalid_request_error"},
		})
		return
	}
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if !imagesOpenAICompatible(provider.Type) || !caps.Image {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "images require an openai or openai_compatible provider with image capability",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeImage(reqBody.Model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not an image model (use dall-e-*, gpt-image-*, or a catalog image route)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	log.Info("images", "model", reqBody.Model, "target_model", route.TargetModel, "provider", provider.ID)
	bound, credID, bindErr := p.bindProviderSecret(ctx, snap, provider, key, policy.Request{
		Model:   reqBody.Model,
		Path:    call.Route.Path,
		Tags:    call.Tags,
		Headers: call.Headers,
	})
	if bindErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": bindErr.Error(), "type": "server_error"},
		})
		return
	}
	client, err := p.imagesBackend(bound.Type)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	resp, err := client.Images(llm.WithExtraHeaders(ctx, call.RequestHeaders), bound, route.TargetModel, body)
	status := "ok"
	imageCount := 0
	if err != nil {
		log.Error("images upstream", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID, CredentialID: credID,
			Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityImage, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: bound.Type, TargetModel: route.TargetModel,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}

	if resp.StatusCode < 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]string{"message": "failed to read upstream", "type": "server_error"},
			})
			status = "error"
			p.recordUsage(UsageEvent{
				OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID, CredentialID: credID,
				Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
				Status: status, LatencyMs: time.Since(start).Milliseconds(),
				Modality: ModalityImage, Tags: cloneTags(call.Tags),
			})
			p.runAfterCall(ctx, snap, call, AfterCallInfo{
				Status: status, LatencyMs: time.Since(start).Milliseconds(),
				ProviderType: bound.Type, TargetModel: route.TargetModel,
			})
			return
		}
		imageCount = countImagesInResponse(respBody)
		wantB64 := strings.EqualFold(strings.TrimSpace(reqBody.ResponseFormat), "b64_json")
		if rewritten, persistErr := p.maybePersistImages(ctx, snap, key, respBody, wantB64); persistErr != nil {
			log.Warn("images persist skipped", "err", persistErr)
		} else if rewritten != nil {
			respBody = rewritten
			imageCount = countImagesInResponse(respBody)
		}
		applyResponseHeaders(w, call)
		for k, vals := range resp.Header {
			if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") ||
				strings.EqualFold(k, "Content-Length") {
				continue
			}
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
	} else {
		applyResponseHeaders(w, call)
		if copyErr := CopyResponse(w, resp); copyErr != nil {
			log.Error("copy images response", "err", copyErr)
			status = "error"
		}
	}

	metrics := map[string]any{}
	if imageCount > 0 {
		metrics["images"] = float64(imageCount)
	}
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID, CredentialID: credID,
		Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityImage, Tags: cloneTags(call.Tags), Metrics: metrics,
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: bound.Type, TargetModel: route.TargetModel,
	})
}

func countImagesInResponse(body []byte) int {
	var parsed struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0
	}
	return len(parsed.Data)
}

// maybePersistImages stores generated assets when the org object store is enabled.
// Returns rewritten body or (nil, nil) when passthrough. Errors are soft (caller may ignore).
func (p *Pipeline) maybePersistImages(ctx context.Context, snap *snapshot.Snapshot, key snapshot.APIKey, body []byte, wantB64 bool) ([]byte, error) {
	cfg := snap.ResolveObjectStore(key.OrganizationID)
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	store, ttl, err := p.openObjectStore(ctx, snap, cfg)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, nil
	}
	return p.rewritePersistedImages(ctx, store, ttl, key, body, wantB64)
}

func (p *Pipeline) rewritePersistedImages(ctx context.Context, store objectstore.Store, ttl time.Duration, key snapshot.APIKey, body []byte, wantB64 bool) ([]byte, error) {
	var parsed struct {
		Created int64 `json:"created"`
		Data    []struct {
			URL     string `json:"url,omitempty"`
			B64JSON string `json:"b64_json,omitempty"`
			Revised string `json:"revised_prompt,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse images response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, nil
	}

	httpClient := p.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	outData := make([]map[string]any, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		raw, contentType, ext, err := imageBytesFromItem(ctx, httpClient, item.URL, item.B64JSON)
		if err != nil {
			return nil, err
		}
		assetID, err := newAssetID()
		if err != nil {
			return nil, err
		}
		objKey := objectstore.AssetKey(key.OrganizationID, key.ProjectID, assetID, ext)
		if err := store.Put(ctx, objKey, bytes.NewReader(raw), int64(len(raw)), objectstore.PutOptions{
			ContentType: contentType,
			Metadata: map[string]string{
				"org_id":     key.OrganizationID,
				"project_id": key.ProjectID,
			},
		}); err != nil {
			return nil, err
		}
		entry := map[string]any{}
		if item.Revised != "" {
			entry["revised_prompt"] = item.Revised
		}
		if wantB64 {
			entry["b64_json"] = base64.StdEncoding.EncodeToString(raw)
		} else {
			url, err := store.PresignGet(ctx, objKey, ttl)
			if err != nil {
				return nil, err
			}
			entry["url"] = url
		}
		outData = append(outData, entry)
	}

	out := map[string]any{"data": outData}
	if parsed.Created != 0 {
		out["created"] = parsed.Created
	}
	return json.Marshal(out)
}

func imageBytesFromItem(ctx context.Context, httpClient *http.Client, url, b64 string) ([]byte, string, string, error) {
	if b64 != "" {
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, "", "", fmt.Errorf("decode b64_json: %w", err)
		}
		ct, ext := sniffImage(raw)
		return raw, ct, ext, nil
	}
	if url == "" {
		return nil, "", "", fmt.Errorf("image item has neither url nor b64_json")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch image url: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", "", fmt.Errorf("fetch image url: status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct, _ = sniffImage(raw)
	}
	_, ext := sniffImage(raw)
	return raw, ct, ext, nil
}

func sniffImage(raw []byte) (contentType, ext string) {
	switch {
	case bytes.HasPrefix(raw, []byte{0x89, 0x50, 0x4e, 0x47}):
		return "image/png", "png"
	case bytes.HasPrefix(raw, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", "jpg"
	case bytes.HasPrefix(raw, []byte("RIFF")) && len(raw) > 11 && bytes.Equal(raw[8:12], []byte("WEBP")):
		return "image/webp", "webp"
	case bytes.HasPrefix(raw, []byte("GIF8")):
		return "image/gif", "gif"
	default:
		return "application/octet-stream", "bin"
	}
}

func newAssetID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (p *Pipeline) openObjectStore(ctx context.Context, snap *snapshot.Snapshot, cfg *snapshot.ObjectStoreConfig) (objectstore.Store, time.Duration, error) {
	accessKey, secretKey, err := p.resolveObjectStoreKeys(ctx, snap, cfg)
	if err != nil {
		return nil, 0, err
	}
	store, err := objectstore.New(objectstore.Config{
		Endpoint:  cfg.Endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    cfg.Region,
		Bucket:    cfg.Bucket,
		UseSSL:    cfg.UseSSL,
		PathStyle: cfg.PathStyle,
	})
	if err != nil {
		return nil, 0, err
	}
	ttl := time.Duration(cfg.PresignTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	return store, ttl, nil
}

func (p *Pipeline) resolveObjectStoreKeys(ctx context.Context, snap *snapshot.Snapshot, cfg *snapshot.ObjectStoreConfig) (accessKey, secretKey string, err error) {
	if cfg.CredentialID != "" {
		cred, ok := snap.Credentials[cfg.CredentialID]
		if !ok {
			return "", "", fmt.Errorf("object store credential %q not found", cfg.CredentialID)
		}
		if p.Credentials == nil {
			return "", "", fmt.Errorf("credential resolver not configured")
		}
		plain, err := p.Credentials.Open(ctx, cred)
		if err != nil {
			return "", "", err
		}
		var parsed struct {
			AccessKey string `json:"access_key"`
			SecretKey string `json:"secret_key"`
		}
		if json.Unmarshal([]byte(plain), &parsed) == nil && parsed.AccessKey != "" && parsed.SecretKey != "" {
			return parsed.AccessKey, parsed.SecretKey, nil
		}
		return "", "", fmt.Errorf("object store credential must be JSON {\"access_key\",\"secret_key\"}")
	}
	if cfg.AccessKeyEnv == "" || cfg.SecretKeyEnv == "" {
		return "", "", fmt.Errorf("object store access_key_env and secret_key_env required")
	}
	if p.Secrets == nil {
		return "", "", fmt.Errorf("secrets resolver not configured")
	}
	accessKey, err = p.Secrets.Get(ctx, cfg.AccessKeyEnv)
	if err != nil {
		return "", "", err
	}
	secretKey, err = p.Secrets.Get(ctx, cfg.SecretKeyEnv)
	if err != nil {
		return "", "", err
	}
	return accessKey, secretKey, nil
}
