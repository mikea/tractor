package tractor

import (
	"fmt"
	"sync"
)

const defaultMailboxSize = 1000

func Start(root SetupHandler) ActorSystem {
	return run(Setup(root))
}

func run(root Behavior) ActorSystem {
	system := &actorSystemImpl{}
	system.Start(root)
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

func (ctx localActorContext) Spawn(setup SetupHandler) ActorRef {
	ref := &localActorRef{
		system:  ctx.system,
		mailbox: make(chan interface{}, defaultMailboxSize),
	}
	ref.spawn(Setup(setup))
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
			currentBehavior = setup.apply(ref.ctx)
		}

		for {
			if _, ok := currentBehavior.(*stoppedBehavior); ok {
				break
			}

			msg := <-ref.mailbox
			var newBehavior Behavior
			switch b := currentBehavior.(type) {
			case *receiveBehavior:
				newBehavior = b.apply(msg)
			default:
				panic(fmt.Sprintf("Bad behavior: %T", currentBehavior))
			}

			if _, ok := newBehavior.(*sameBehavior); !ok {
				currentBehavior = newBehavior
			}
		}

		ref.system.waitGroup.Done()
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
