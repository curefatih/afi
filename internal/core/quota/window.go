package quota

type Window string

const (
	WindowMinute Window = "MINUTE"

	WindowHour Window = "HOUR"

	WindowDay Window = "DAY"

	WindowMonth Window = "MONTH"

	WindowLifetime Window = "LIFETIME"
)