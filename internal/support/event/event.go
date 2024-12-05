package event

import (
	"github.com/asaskevich/EventBus"

	"github.com/kuvalkin/gophermart-loyalty/internal/support/log"
)

var (
	bus = EventBus.New()
)

func Subscribe(topic string, fn any) error {
	return bus.SubscribeAsync(topic, fn, false)
}

func Publish(topic string, args ...interface{}) {
	log.Logger().Named("event").Debugw(topic, "args", args)

	bus.Publish(topic, args...)
}

func Release() {
	bus.WaitAsync()
}
