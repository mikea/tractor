package tractor

type Behavior interface {
	tag()
}

type SetupHandler func(ctx ActorContext) Behavior

type receiveBehavior struct {
	handler func(msg interface{}) Behavior
}

func (r receiveBehavior) tag() {
	panic("implement me")
}

func Receive(handler func(msg interface{}) Behavior) Behavior {
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

type sameBehavior struct{}

func (s sameBehavior) tag() {
	panic("implement me")
}

type stoppedBehavior struct{}

func (s stoppedBehavior) tag() {
	panic("implement me")
}

func Same() Behavior {
	return &sameBehavior{}
}

func Stopped() Behavior {
	return &stoppedBehavior{}
}
