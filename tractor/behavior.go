package tractor

import "reflect"

type behavior interface {
	tag()
}

type receiveBehavior struct {
	handler func(msg interface{}) MessageHandler
}

func (r receiveBehavior) tag() {
	panic("implement me")
}

func (r receiveBehavior) apply(msg interface{}) behavior {
	return receive(r.handler(msg))
}

func setup(handler SetupHandler) behavior {
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

func (s setupBehavior) apply(ctx ActorContext) behavior {
	return receive(s.handler(ctx))
}

func receive(handler MessageHandler) behavior {
	if handler == nil {
		return &sameBehavior{}
	}
	if isStopped(handler) {
		return &stoppedBehavior{}
	}
	return &receiveBehavior{
		handler: handler,
	}
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

func (*stoppedBehavior) handle(_ interface{}) MessageHandler {
	panic("should never be called")
}

func isStopped(handler MessageHandler) bool {
	// todo: faster?
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(Stopped()).Pointer()
}
