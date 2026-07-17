package provider

type Resource struct {
	URL string

	Data []byte

	MimeType string

	Name string
}

type Image struct {
	Resource
}

type Audio struct {
	Resource
}
