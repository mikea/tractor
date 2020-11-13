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
}

type MessageHandler func(message interface{}) MessageHandler
type SetupHandler func(ctx ActorContext) MessageHandler

type PostInitSignal struct{}
type PostStopSignal struct{}

func Stopped() MessageHandler {
	return stopped.handle
}
