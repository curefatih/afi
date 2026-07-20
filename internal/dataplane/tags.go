package dataplane

import (
	"net/http"
	"strings"
)

// AFITagsHeader is the request header for external user tags.
const AFITagsHeader = "X-AFI-Tags"

// ParseAFITags parses "key:value,key:value" tag headers.
// Pairs are comma-separated; each pair splits on the first ':'.
// Keys and values are trimmed; empty keys are skipped; last duplicate key wins.
func ParseAFITags(header string) map[string]string {
	header = strings.TrimSpace(header)
	if header == "" {
		return map[string]string{}
	}
	out := make(map[string]string)
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		out[key] = strings.TrimSpace(val)
	}
	return out
}

// TagsFromRequest reads X-AFI-Tags from the request.
func TagsFromRequest(r *http.Request) map[string]string {
	if r == nil {
		return map[string]string{}
	}
	return ParseAFITags(r.Header.Get(AFITagsHeader))
}

func cloneTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}
