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
	self           *localActorRef
	parent         *localActorRef
	deliverSignals bool
}

func newContext(system *actorSystemImpl, self *localActorRef, parent *localActorRef) *localActorContext {
	return &localActorContext{
		system: system,
		self:   self,
		parent: parent,
	}
}

func (ctx *localActorContext) Parent() ActorRef {
	return ctx.parent
}

func (ctx *localActorContext) Self() ActorRef {
	return ctx.self
}

func (ctx *localActorContext) DeliverSignals(value bool) {
	ctx.deliverSignals = value
}

func (ctx *localActorContext) Spawn(handler SetupHandler) ActorRef {
	ref := &localActorRef{
		system:  ctx.system,
		mailbox: make(chan interface{}, defaultMailboxSize),
	}
	ref.start(ctx.self, setup(handler))
	return ref
}

func (ref *localActorRef) start(parent *localActorRef, root behavior) {
	ref.system.waitGroup.Add(1)

	go func() {
		ctx := newContext(ref.system, ref, parent)
		ref.ctx = ctx

		currentBehavior := root
		if setup, ok := currentBehavior.(*setupBehavior); ok {
			currentBehavior = setup.apply(ref.ctx)
		}

		if ctx.deliverSignals {
			currentBehavior = ref.deliver(currentBehavior, PostInitSignal{})
		}

		for {
			if _, ok := currentBehavior.(*stoppedBehavior); ok {
				break
			}
			msg := <-ref.mailbox
			currentBehavior = ref.deliver(currentBehavior, msg)
		}

		if ctx.deliverSignals {
			switch b := currentBehavior.(type) {
			case *stoppedBehavior:
				b.handler(PostStopSignal{})
			default:
				panic(fmt.Sprintf("Expected stopped behavior: %T", currentBehavior))
			}
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
	system.root.start(nil, root)
}
