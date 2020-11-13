package tractor

type ActorSystem interface {
	Root() ActorRef
	Wait()
}

type ActorRef interface {
	Tell(msg interface{})
}

type ActorContext interface {
	Spawn(setup SetupHandler) ActorRef
	DeliverSignals(value bool)
	Parent() ActorRef
	Self() ActorRef
	Children() []ActorRef
}

type PostInitSignal struct{}
type PostStopSignal struct{}

type MessageHandler func(message interface{}) MessageHandler
type SetupHandler func(ctx ActorContext) MessageHandler

func Stopped() MessageHandler {
	return stopped.handle
}
