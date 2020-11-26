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
	Watch(actor ActorRef)
}

type PostInitSignal struct{}
type PreStopSignal struct{}
type PostStopSignal struct{}

type Terminated struct {
	Ref ActorRef
}

type SetupHandler func(ctx ActorContext) MessageHandler
type MessageHandler func(message interface{}) MessageHandler

func Stopped() MessageHandler {
	return stopped.handle
}
