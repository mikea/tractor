package tractor

func Handle(handler func(msg interface{}) Behavior) Behavior {
	return &behaviorImpl{
		Handler: handler,
	}
}

var SAME = &behaviorImpl{}
var STOPPED = &behaviorImpl{}

func Same() Behavior {
	return SAME
}

func Stopped() Behavior {
	return STOPPED
}
