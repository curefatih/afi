package event

import "time"

type Event struct {
	ID        string
	Type      string
	Timestamp time.Time

	Payload any
}
