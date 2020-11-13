package tractor

type ActorSystem interface {
	Root() ActorRef
	Wait()
}

type ActorRef interface {
	Tell(msg interface{})
}

type ActorContext interface {
	Spawn(behavior Behavior) ActorRef
}
