package stages

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/event"
)

type PublishEvents struct {
	publisher event.Publisher
}

func NewPublishEvents(
	publisher event.Publisher,
) *PublishEvents {
	return &PublishEvents{
		publisher: publisher,
	}
}

func (s *PublishEvents) Name() string {
	return "publish_events"
}

func (s *PublishEvents) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	if state.Response == nil {
		return nil
	}

	event := event.Event{
		ID:        uuid.NewString(),
		Type:      "gateway.request.completed",
		Timestamp: time.Now().UTC(),
		Payload: gateway.RequestCompleted{
			RequestID: state.RequestID().String(),

			Principal: *state.Principal(),

			Provider: state.Route().Target.ProviderID,

			Model: state.Model().ID,

			Usage: *state.Usage(),

			Cost: *state.Cost(),

			ResponseID: state.Response().ID,

			FinishReason: state.Response().FinishReason,
		},
	}

	return s.publisher.Publish(
		ctx,
		event,
	)
}
