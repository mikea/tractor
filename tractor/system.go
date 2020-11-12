package tractor

import (
	"fmt"
	"sync"
)

const defaultMailboxSize = 1000

func Start(handler SetupHandler) ActorSystem {
	system := &actorSystemImpl{}
	system.Start(Setup(handler))
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
	ctx     ActorContext
}

func (ref *localActorRef) Tell(msg interface{}) {
	ref.mailbox <- msg
}

type localActorContext struct {
	system *actorSystemImpl
}

func (ctx localActorContext) Spawn(handler SetupHandler) ActorRef {
	ref := &localActorRef{
		system:  ctx.system,
		mailbox: make(chan interface{}, defaultMailboxSize),
	}
	ref.spawn(Setup(handler))
	return ref
}

func (ref *localActorRef) spawn(root Behavior) {
	ref.system.waitGroup.Add(1)

	go func() {
		ref.ctx = &localActorContext{
			system: ref.system,
		}

		currentBehavior := root
		if setup, ok := currentBehavior.(*setupBehavior); ok {
			currentBehavior = setup.handler(ref.ctx)
		}
		for {
			msg := <-ref.mailbox
			var newBehavior Behavior
			switch b := currentBehavior.(type) {
			case *receiveBehavior:
				newBehavior = b.handler(msg)
			default:
				panic(fmt.Sprintf("Bad behavior: %t", currentBehavior))
			}
			switch newBehavior.(type) {
			case *stoppedBehavior:
				ref.system.waitGroup.Done()
				return
			case *sameBehavior:
				continue
			default:
				currentBehavior = newBehavior
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
