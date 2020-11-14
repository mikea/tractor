package tractor

import (
	"fmt"
	"sync"
)

const defaultMailboxSize = 1000

func Start(root SetupHandler) ActorSystem {
	system := &actorSystemImpl{}
	system.Start(root)
	return system
}

type actorSystemImpl struct {
	topCotext *localActorContext
	root      *localActorRef
}

func (system *actorSystemImpl) Wait() {
	system.topCotext.childrenWaitGroup.Wait()
}

type localActorRef struct {
	system   *actorSystemImpl
	mailbox  chan interface{}
	commands chan interface{}
	ctx      *localActorContext
}

func (ref *localActorRef) Tell(msg interface{}) {
	ref.mailbox <- msg
}

type localActorContext struct {
	system            *actorSystemImpl
	parent            *localActorContext
	childrenWaitGroup *sync.WaitGroup
	self              *localActorRef
	deliverSignals    bool
	children          []*localActorRef
}

func (ctx *localActorContext) Children() []ActorRef {
	result := make([]ActorRef, len(ctx.children))
	for i, ref := range ctx.children {
		result[i] = ref
	}
	return result
}

func newContext(system *actorSystemImpl, self *localActorRef, parent *localActorContext) *localActorContext {
	return &localActorContext{
		system:            system,
		self:              self,
		parent:            parent,
		childrenWaitGroup: &sync.WaitGroup{},
	}
}

func (ctx *localActorContext) Parent() ActorRef {
	return ctx.parent.self
}

func (ctx *localActorContext) Self() ActorRef {
	return ctx.self
}

func (ctx *localActorContext) DeliverSignals(value bool) {
	ctx.deliverSignals = value
}

func (ctx *localActorContext) Spawn(handler SetupHandler) ActorRef {
	return ctx.spawn(handler)
}

func (ctx *localActorContext) spawn(handler SetupHandler) *localActorRef {
	ref := &localActorRef{
		system:  ctx.system,
		mailbox: make(chan interface{}, defaultMailboxSize),
	}
	ctx.children = append(ctx.children, ref)
	ctx.childrenWaitGroup.Add(1)
	go func() {
		ref.mainLoop(ctx, setup(handler))
	}()
	return ref
}

type terminate struct{}

func (ref *localActorRef) mainLoop(parentContext *localActorContext, root behavior) {
	ctx := newContext(ref.system, ref, parentContext)
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

		select {
		case msg := <-ref.mailbox:
			currentBehavior = ref.deliver(currentBehavior, msg)
		case cmd := <-ref.commands:
			switch cmd.(type) {
			case *terminate:
				currentBehavior = &stoppedBehavior{}
			default:
				panic(fmt.Sprintf("Bad command: %T", cmd))
			}
		}
	}

	if ctx.deliverSignals {
		switch b := currentBehavior.(type) {
		case *stoppedBehavior:
			b.handler(PreStopSignal{})
		default:
			panic(fmt.Sprintf("Expected stopped behavior: %T", currentBehavior))
		}
	}

	for _, child := range ctx.children {
		child.commands <- &terminate{}
	}
	ctx.childrenWaitGroup.Wait()

	if ctx.deliverSignals {
		switch b := currentBehavior.(type) {
		case *stoppedBehavior:
			b.handler(PostStopSignal{})
		default:
			panic(fmt.Sprintf("Expected stopped behavior: %T", currentBehavior))
		}
	}

	ctx.parent.childrenWaitGroup.Done()
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

func (system *actorSystemImpl) Start(root SetupHandler) {
	system.topCotext = newContext(system, nil, nil)
	system.root = system.topCotext.spawn(root)
}
