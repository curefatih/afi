package plugin

type Plugin struct {
	ID string

	Name string

	Version string

	Enabled bool

	Hooks []Hook
}
