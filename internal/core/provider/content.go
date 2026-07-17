package provider

type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
	ContentTypeAudio ContentType = "audio"
)

type Content interface {
	Type() ContentType
}

type TextContent struct {
	Text string
}

func (TextContent) Type() ContentType {
	return ContentTypeText
}

type ImageContent struct {
	URL string

	MimeType string
}

func (ImageContent) Type() ContentType {
	return ContentTypeImage
}

type AudioContent struct {
	URL string

	MimeType string
}

func (AudioContent) Type() ContentType {
	return ContentTypeAudio
}
