package tractor

import "sync"

func Start(behavior Behavior) ActorSystem {
	system := &actorSystemImpl{}
	system.Start(behavior)
	return system
}

type actorSystemImpl struct {
	root      *localActorRef
	waitGroup *sync.WaitGroup
}

func (system *actorSystemImpl) Wait() {
	system.waitGroup.Wait()
}

type localActorRef struct {
	system  *actorSystemImpl
	mailbox chan interface{}
}

func (ref *localActorRef) Tell(msg interface{}) {
	ref.mailbox <- msg
}

func (ref *localActorRef) spawn(root Behavior) {
	ref.system.waitGroup.Add(1)

	go func() {
		currentBehavior := root
		for {
			msg := <-ref.mailbox
			newHandler := currentBehavior.Handler(msg)
			if newHandler == STOPPED {
				ref.system.waitGroup.Done()
				return
			}
			if newHandler != SAME {
				currentBehavior = newHandler
			}
		}
	}()
}

func (system *actorSystemImpl) Root() ActorRef {
	return system.root
}

func (system *actorSystemImpl) Start(root Behavior) {
	system.waitGroup = &sync.WaitGroup{}

	system.root = &localActorRef{
		system:  system,
		mailbox: make(chan interface{}, 1000),
	}
	system.root.spawn(root)
}
