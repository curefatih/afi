package usage

// Modality identifies the request surface that produced usage.
type Modality string

const (
	ModalityChat     Modality = "chat"
	ModalityMessages Modality = "messages"
	ModalityTTS      Modality = "tts"
	ModalitySTT      Modality = "stt"
	ModalityMCP      Modality = "mcp"
	ModalityA2A      Modality = "a2a"
)

func (m Modality) String() string {
	if m == "" {
		return string(ModalityChat)
	}
	return string(m)
}

// NormalizeModality defaults empty modality to chat.
func NormalizeModality(m string) string {
	if m == "" {
		return string(ModalityChat)
	}
	return m
}
