package llm

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
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

// rewriteMultipartModel rebuilds a multipart body with the model field set to targetModel.
func rewriteMultipartModel(contentType string, body io.Reader, targetModel string) ([]byte, string, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", fmt.Errorf("content-type: %w", err)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, "", fmt.Errorf("multipart boundary missing")
	}
	mr := multipart.NewReader(body, boundary)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	modelWritten := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		name := part.FormName()
		hdr := textproto.MIMEHeader{}
		for k, vals := range part.Header {
			for _, v := range vals {
				hdr.Add(k, v)
			}
		}
		w, err := mw.CreatePart(hdr)
		if err != nil {
			return nil, "", err
		}
		if name == "model" {
			if _, err := io.WriteString(w, targetModel); err != nil {
				return nil, "", err
			}
			modelWritten = true
			_, _ = io.Copy(io.Discard, part)
			continue
		}
		if _, err := io.Copy(w, part); err != nil {
			return nil, "", err
		}
	}
	if !modelWritten {
		w, err := mw.CreateFormField("model")
		if err != nil {
			return nil, "", err
		}
		if _, err := io.WriteString(w, targetModel); err != nil {
			return nil, "", err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), mw.FormDataContentType(), nil
}
