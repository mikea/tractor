package tractor

import "reflect"

type behavior interface {
}

type receiveBehavior struct {
	handler func(msg interface{}) MessageHandler
}

func (r *receiveBehavior) apply(msg interface{}) behavior {
	newHandler := r.handler(msg)
	if newHandler == nil {
		return &sameBehavior{}
	}
	if isStopped(newHandler) {
		return &stoppedBehavior{
			handler: r.handler,
		}
	}
	return &receiveBehavior{
		handler: newHandler,
	}
}

func setup(handler SetupHandler) behavior {
	return &setupBehavior{
		handler: handler,
	}
}

type setupBehavior struct {
	handler SetupHandler
}

func (s setupBehavior) apply(ctx ActorContext) behavior {
	handler := s.handler(ctx)
	if handler == nil {
		panic("nil result of setup handler")
	}
	if isStopped(handler) {
		return &stoppedBehavior{}
	}
	return &receiveBehavior{
		handler: handler,
	}
}

type sameBehavior struct{}

var stopped stoppedBehavior

type stoppedBehavior struct {
	handler MessageHandler
}

func (s *stoppedBehavior) handle(_ interface{}) MessageHandler {
	panic("should not be called")
}

func isStopped(handler MessageHandler) bool {
	// todo: faster?
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(Stopped()).Pointer()
}
