package provider

type ResponseFormatType string

const (
	ResponseFormatText       ResponseFormatType = "TEXT"
	ResponseFormatJSON       ResponseFormatType = "JSON"
	ResponseFormatJSONSchema ResponseFormatType = "JSON_SCHEMA"
)

type ResponseFormat struct {
	Type ResponseFormatType

	// Used when Type == JSON_SCHEMA
	Schema *JSONSchema
}