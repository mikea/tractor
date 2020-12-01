package tractor

type ActorSystem interface {
	Root() ActorRef
	Context() ActorContext
	Wait()
}

type ActorRef interface {
	Tell(ctx ActorContext, msg interface{})
}

type ActorContext interface {
	Parent() ActorRef
	Self() ActorRef
	Sender() ActorRef

	Children() []ActorRef
	Spawn(setup SetupHandler) ActorRef
	Watch(actor ActorRef)
	WatchWith(actor ActorRef, msg interface{})

	DeliverSignals(value bool)
	Ask(ref ActorRef, msg interface{}) chan interface{}
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
