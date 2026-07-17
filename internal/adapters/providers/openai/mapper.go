package openai

import (
	"encoding/json"
	"io"
)

func DecodeResponse(r io.Reader) (*ChatCompletionResponse, error) {

	var resp ChatCompletionResponse

	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
