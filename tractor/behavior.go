package tractor

import "reflect"

var stopped stoppedBehavior

type stoppedBehavior struct {
}

func (s *stoppedBehavior) handle(_ interface{}) MessageHandler {
	panic("should not be called")
}

func isStopped(handler MessageHandler) bool {
	// todo: can this be made faster?
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(Stopped()).Pointer()
}
