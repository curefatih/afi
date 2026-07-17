package event

type Registry interface {
	Subscribe(
		event string,
		subscriber Subscriber,
	)

	Subscribers(
		event string,
	) []Subscriber
}
