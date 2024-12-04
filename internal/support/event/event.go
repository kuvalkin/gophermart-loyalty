package event

import "github.com/asaskevich/EventBus"

var (
	bus = EventBus.New()
)

func Subscribe(topic string, fn any) error {
	return bus.SubscribeAsync(topic, fn, false)
}

func Publish(topic string, args ...interface{}) {
	bus.Publish(topic, args...)
}

func Release() {
	bus.WaitAsync()
}
