package provider

type ModerationRequest struct {
	Input []string
}

func (ModerationRequest) Capability() Capability {
	return CapabilityModeration
}