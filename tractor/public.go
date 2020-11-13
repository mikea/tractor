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
}

type MessageHandler func(message interface{}) MessageHandler
type SetupHandler func(ctx ActorContext) MessageHandler
