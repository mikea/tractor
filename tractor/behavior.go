package tractor

import "reflect"

type Behavior interface {
	tag()
}

type receiveBehavior struct {
	handler func(msg interface{}) MessageHandler
}

func (r receiveBehavior) tag() {
	panic("implement me")
}

func (r receiveBehavior) apply(msg interface{}) Behavior {
	return receive(r.handler(msg))
}

func Receive(handler MessageHandler) Behavior {
	return &receiveBehavior{
		handler: handler,
	}
}

func Setup(handler SetupHandler) Behavior {
	return &setupBehavior{
		handler: handler,
	}
}

type setupBehavior struct {
	handler SetupHandler
}

func (s setupBehavior) tag() {
	panic("implement me")
}

func (s setupBehavior) apply(ctx ActorContext) Behavior {
	return receive(s.handler(ctx))
}

func receive(handler MessageHandler) Behavior {
	if handler == nil {
		return &sameBehavior{}
	}
	if IsStopped(handler) {
		return &stoppedBehavior{}
	}
	return Receive(handler)
}

type sameBehavior struct{}

func (s sameBehavior) tag() {
	panic("implement me")
}

type stoppedBehavior struct{}

func (s stoppedBehavior) tag() {
	panic("implement me")
}

var stopped stoppedBehavior

func Stopped() MessageHandler {
	return stopped.handle
}

func (*stoppedBehavior) handle(_ interface{}) MessageHandler {
	panic("should never be called")
}

func IsStopped(handler MessageHandler) bool {
	// todo: faster?
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(Stopped()).Pointer()
}
