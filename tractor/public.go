package tractor

type ActorSystem interface {
	Root() ActorRef
	Wait()
}

type ActorRef interface {
	Tell(msg interface{})
}

type behaviorImpl struct {
	Handler func(msg interface{}) Behavior
}

type Behavior *behaviorImpl
