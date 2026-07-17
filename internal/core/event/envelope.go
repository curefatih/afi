package event

import "time"

type Envelope struct {
	ID string

	Time time.Time

	CorrelationID string

	Event Event
}
