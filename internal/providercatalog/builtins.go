package providercatalog

func init() {
	RegisterSpec(Spec{
		Type:             "openai",
		DisplayName:      "OpenAI",
		DefaultBaseURL:   "https://api.openai.com/v1",
		DefaultAPIKeyEnv: "OPENAI_API_KEY",
		Capabilities:     Capabilities{Chat: true, Stream: true, TTS: true, STT: true, Embedding: true, Image: true},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_openai",
	})
	RegisterSpec(Spec{
		Type:             "openai_compatible",
		DisplayName:      "Ollama / compatible",
		DefaultBaseURL:   "http://127.0.0.1:11434/v1",
		DefaultAPIKeyEnv: "OLLAMA_API_KEY",
		Capabilities:     Capabilities{Chat: true, Stream: true, TTS: true, STT: true, Embedding: true, Image: true},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_ollama",
		CatalogAlias:     "openai",
	})
	RegisterSpec(Spec{
		Type:             "anthropic",
		DisplayName:      "Anthropic",
		DefaultBaseURL:   "https://api.anthropic.com/v1",
		DefaultAPIKeyEnv: "ANTHROPIC_API_KEY",
		Capabilities:     Capabilities{Chat: true, Stream: true},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_anthropic",
	})
	RegisterSpec(Spec{
		Type:             "gemini",
		DisplayName:      "Gemini",
		DefaultBaseURL:   "https://generativelanguage.googleapis.com/v1beta",
		DefaultAPIKeyEnv: "GEMINI_API_KEY",
		Capabilities:     Capabilities{Chat: true, Stream: true},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_gemini",
	})
	RegisterSpec(Spec{
		Type:             "bedrock",
		DisplayName:      "AWS Bedrock",
		DefaultBaseURL:   "https://bedrock-runtime.us-west-2.amazonaws.com",
		DefaultAPIKeyEnv: "",
		Capabilities:     Capabilities{Chat: true, Stream: true},
		AuthMode:         AuthOptional,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_bedrock",
	})
	RegisterSpec(Spec{
		Type:             "elevenlabs",
		DisplayName:      "ElevenLabs",
		DefaultBaseURL:   "https://api.elevenlabs.io",
		DefaultAPIKeyEnv: "ELEVENLABS_API_KEY",
		Capabilities:     Capabilities{TTS: true, STT: true},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_elevenlabs",
	})
	RegisterSpec(Spec{
		Type:             "echo",
		DisplayName:      "Echo (extension)",
		DefaultBaseURL:   "http://localhost/echo",
		DefaultAPIKeyEnv: "ECHO_UNUSED",
		Capabilities:     Capabilities{Chat: true, Stream: false},
		AuthMode:         AuthAPIKey,
		UIVisible:        true,
		Seed:             true,
		SeedID:           "prov_echo",
		SeedRoute: &SeedRoute{
			ID:          "route_echo",
			Model:       "echo-demo",
			TargetModel: "echo-demo",
		},
	})
}
