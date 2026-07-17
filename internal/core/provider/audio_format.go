package provider

type AudioFormat string

const (
	AudioFormatMP3 AudioFormat = "MP3"

	AudioFormatWAV AudioFormat = "WAV"

	AudioFormatFLAC AudioFormat = "FLAC"

	AudioFormatPCM AudioFormat = "PCM"

	AudioFormatOPUS AudioFormat = "OPUS"

	AudioFormatAAC AudioFormat = "AAC"
)