package tractor

import (
	"fmt"
	"sync"
)

const defaultMailboxSize = 1000
const defaultCommandsSize = 2

func Start(root SetupHandler) ActorSystem {
	system := &actorSystemImpl{}
	system.start(root)
	return system
}

type actorSystemImpl struct {
	topContext *localActorContext
	root       *localActorRef
}

func (system *actorSystemImpl) Wait() {
	system.topContext.childrenWaitGroup.Wait()
}

type localActorRef struct {
	system   *actorSystemImpl
	mailbox  chan interface{}
	commands chan interface{}
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
	listeners         []ActorRef
}

func (ctx *localActorContext) Watch(actor ActorRef) {
	actor.(*localActorRef).commands <- &listenCommand{ref: ctx.self}
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
		system:   ctx.system,
		mailbox:  make(chan interface{}, defaultMailboxSize),
		commands: make(chan interface{}, defaultCommandsSize),
	}
	ctx.children = append(ctx.children, ref)
	ctx.childrenWaitGroup.Add(1)
	go func() {
		newContext(ref.system, ref, ctx).
			mainLoop(ref, handler)
	}()
	return ref
}

type terminateCommand struct{}
type listenCommand struct {
	ref ActorRef
}

func (ctx *localActorContext) mainLoop(ref *localActorRef, setup SetupHandler) {
	messageHandler := setup(ctx)
	if messageHandler == nil {
		messageHandler = ignoreAll
	}

	lastMessageHandler := messageHandler
	if ctx.deliverSignals && !isStopped(messageHandler) {
		if newHandler := messageHandler(PostInitSignal{}); newHandler != nil {
			messageHandler = newHandler
		}
	}

	for {
		if isStopped(messageHandler) {
			break
		}

		lastMessageHandler = messageHandler
		select {
		case cmd := <-ref.commands:
			switch command := cmd.(type) {
			case *terminateCommand:
				messageHandler = Stopped()
			case *listenCommand:
				ctx.listeners = append(ctx.listeners, command.ref)
			default:
				panic(fmt.Sprintf("Bad command: %T", cmd))
			}
		case msg := <-ref.mailbox:
			if newHandler := messageHandler(msg); newHandler != nil {
				messageHandler = newHandler
			}
		}
	}

	// drain commands if any
l:
	for {
		select {
		case cmd := <-ref.commands:
			switch command := cmd.(type) {
			case *terminateCommand:
				// do nothing
			case *listenCommand:
				ctx.listeners = append(ctx.listeners, command.ref)
			default:
				panic(fmt.Sprintf("Bad command: %T", cmd))
			}
		default:
			break l
		}
	}

	if ctx.deliverSignals {
		lastMessageHandler(PreStopSignal{})
	}

	for _, child := range ctx.children {
		child.commands <- &terminateCommand{}
	}
	ctx.childrenWaitGroup.Wait()

	if ctx.deliverSignals {
		lastMessageHandler(PostStopSignal{})
	}
	ctx.parent.childrenWaitGroup.Done()

	for _, listener := range ctx.listeners {
		listener.Tell(Terminated{Ref: ctx.self})
	}
}

func (system *actorSystemImpl) Root() ActorRef {
	return system.root
}

func (system *actorSystemImpl) start(root SetupHandler) {
	system.topContext = newContext(system, nil, nil)
	system.root = system.topContext.spawn(root)
}
