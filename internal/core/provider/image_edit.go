package provider

type ImageEditRequest struct {
	Images []Image

	Mask *Image

	Prompt string

	Count int

	ResponseFormat ImageResponseFormat
}

func (ImageEditRequest) Capability() Capability {
	return CapabilityImageEdit
}