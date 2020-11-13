package tractor

import (
	"fmt"
	"sync"
)

const defaultMailboxSize = 1000

func Start(root SetupHandler) ActorSystem {
	return run(setup(root))
}

func run(root behavior) ActorSystem {
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
	ctx     *localActorContext
}

func (ref *localActorRef) Tell(msg interface{}) {
	ref.mailbox <- msg
}

type localActorContext struct {
	system         *actorSystemImpl
	deliverSignals bool
}

func (ctx *localActorContext) DeliverSignals(value bool) {
	ctx.deliverSignals = value
}

func (ctx *localActorContext) Spawn(handler SetupHandler) ActorRef {
	ref := &localActorRef{
		system:  ctx.system,
		mailbox: make(chan interface{}, defaultMailboxSize),
	}
	ref.spawn(setup(handler))
	return ref
}

func (ref *localActorRef) spawn(root behavior) {
	ref.system.waitGroup.Add(1)

	go func() {
		ctx := &localActorContext{
			system: ref.system,
		}
		ref.ctx = ctx

		currentBehavior := root
		if setup, ok := currentBehavior.(*setupBehavior); ok {
			currentBehavior = setup.apply(ref.ctx)
		}

		if ctx.deliverSignals {
			currentBehavior = ref.deliver(currentBehavior, postInit)
		}

		for {
			if _, ok := currentBehavior.(*stoppedBehavior); ok {
				break
			}

			msg := <-ref.mailbox
			currentBehavior = ref.deliver(currentBehavior, msg)
		}

		ref.system.waitGroup.Done()
	}()
}

func (ref *localActorRef) deliver(currentBehavior behavior, msg interface{}) behavior {
	var newBehavior behavior
	switch b := currentBehavior.(type) {
	case *receiveBehavior:
		newBehavior = b.apply(msg)
	default:
		panic(fmt.Sprintf("Bad behavior: %T", currentBehavior))
	}
	if _, ok := newBehavior.(*sameBehavior); !ok {
		return newBehavior
	}
	return currentBehavior
}

func (system *actorSystemImpl) Root() ActorRef {
	return system.root
}

func (system *actorSystemImpl) Start(root behavior) {
	system.waitGroup = &sync.WaitGroup{}

	system.root = &localActorRef{
		system:  system,
		mailbox: make(chan interface{}, 1000),
	}
	system.root.spawn(root)
}

type postInitStruct struct {
}

func (p postInitStruct) postInit() string {
	panic("postInit")
}

func (p postInitStruct) signal() string {
	panic("signal")
}

var postInit PostInitSignal = &postInitStruct{}
