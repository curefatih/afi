package dataplane

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
)

func multipartFormValue(contentType string, body io.Reader, field string) (string, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return "", fmt.Errorf("multipart boundary missing")
	}
	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if part.FormName() == field {
			b, err := io.ReadAll(part)
			return string(b), err
		}
		_, _ = io.Copy(io.Discard, part)
	}
}
