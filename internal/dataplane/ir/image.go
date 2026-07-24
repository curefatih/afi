package ir

import "strings"

// imageSourceToDataURL renders an ImageSource as a string usable in an OpenAI
// image_url field: the original URL, or a data: URL for inline base64 bytes.
func imageSourceToDataURL(img *ImageSource) string {
	if img == nil {
		return ""
	}
	if img.URL != "" {
		return img.URL
	}
	if img.Data == "" {
		return ""
	}
	mt := img.MediaType
	if mt == "" {
		mt = "application/octet-stream"
	}
	return "data:" + mt + ";base64," + img.Data
}

// parseImageURL turns an OpenAI image_url string into an ImageSource. A data:
// URL is decoded into inline base64 bytes; any other value is kept as a URL.
func parseImageURL(raw string) *ImageSource {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if !strings.HasPrefix(raw, "data:") {
		return &ImageSource{URL: raw}
	}
	rest := strings.TrimPrefix(raw, "data:")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return &ImageSource{URL: raw}
	}
	meta := rest[:comma]
	data := rest[comma+1:]
	mediaType := meta
	if semi := strings.IndexByte(meta, ';'); semi >= 0 {
		mediaType = meta[:semi]
	}
	return &ImageSource{MediaType: mediaType, Data: data}
}
