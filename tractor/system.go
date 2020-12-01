package tractor

import (
	"fmt"
	"os"
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
	context *localActorContext
	root    *localActorRef
}

func (system *actorSystemImpl) Context() ActorContext {
	return system.context
}

func (system *actorSystemImpl) Wait() {
	system.context.childrenWaitGroup.Wait()
}

type localActorRef struct {
	context *localActorContext
}

func (ref *localActorRef) Tell(ctx ActorContext, msg interface{}) {
	ref.context.mailbox <- envelope{msg: msg, sender: ctx.Self()}
}

func (ref *localActorRef) tellCommand(command interface{}) {
	ref.context.commands <- command
}

type terminateListener struct {
	ref ActorRef
	msg interface{}
}

type envelope struct {
	sender ActorRef
	msg    interface{}
}

type localActorContext struct {
	system            *actorSystemImpl
	parent            *localActorContext
	childrenWaitGroup *sync.WaitGroup
	self              *localActorRef
	deliverSignals    bool
	children          []*localActorRef
	listeners         []terminateListener
	mailbox           chan envelope
	commands          chan interface{}
	currentEnvelope   *envelope
}

func (ctx *localActorContext) Ask(ref ActorRef, msg interface{}) chan interface{} {
	ch := make(chan interface{}, 1)
	ctx.Spawn(func(ctx ActorContext) MessageHandler {
		ref.Tell(ctx, msg)
		return func(message interface{}) MessageHandler {
			ch <- message
			return Stopped()
		}
	})
	return ch
}

func (ctx *localActorContext) Sender() ActorRef {
	return ctx.currentEnvelope.sender
}

func (ctx *localActorContext) Watch(actor ActorRef) {
	ctx.WatchWith(actor, Terminated{})
}

func (ctx *localActorContext) WatchWith(actor ActorRef, msg interface{}) {
	actor.(*localActorRef).context.commands <- &listenCommand{ref: ctx.self, msg: msg}
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
		mailbox:           make(chan envelope, defaultMailboxSize),
		commands:          make(chan interface{}, defaultCommandsSize),
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
	ref := &localActorRef{}
	childContext := newContext(ctx.system, ref, ctx)
	ref.context = childContext
	ctx.children = append(ctx.children, ref)
	ctx.childrenWaitGroup.Add(1)
	go func() {
		childContext.mainLoop(handler)
	}()
	return ref
}

type terminateCommand struct{}
type listenCommand struct {
	ref ActorRef
	msg interface{}
}
type childTerminatedCommand struct {
	ref ActorRef
}

func (ctx *localActorContext) mainLoop(setup SetupHandler) {
	messageHandler := ctx.setup(setup)
	if messageHandler == nil {
		messageHandler = Stopped()
	}

	lastMessageHandler := messageHandler
	if ctx.deliverSignals && !isStopped(messageHandler) {
		if newHandler := ctx.deliver(messageHandler, PostInitSignal{}); newHandler != nil {
			messageHandler = newHandler
		}
	}

	for {
		if isStopped(messageHandler) {
			break
		}

		lastMessageHandler = messageHandler
		select {
		case cmd := <-ctx.commands:
			switch command := cmd.(type) {
			case *terminateCommand:
				messageHandler = Stopped()
			case *listenCommand:
				ctx.onListenCommand(command)
			case *childTerminatedCommand:
				ctx.onChildTerminatedCommand(command)
			default:
				panic(fmt.Sprintf("Bad command: %T", cmd))
			}
		case env := <-ctx.mailbox:
			ctx.currentEnvelope = &env
			if newHandler := ctx.deliver(messageHandler, env.msg); newHandler != nil {
				messageHandler = newHandler
			}
		}
	}

	// drain commands if any
l:
	for {
		select {
		case cmd := <-ctx.commands:
			switch command := cmd.(type) {
			case *terminateCommand:
				// do nothing
			case *listenCommand:
				ctx.onListenCommand(command)
			case *childTerminatedCommand:
				ctx.onChildTerminatedCommand(command)
			default:
				panic(fmt.Sprintf("Bad command: %T", cmd))
			}
		default:
			break l
		}
	}

	if ctx.deliverSignals {
		ctx.deliver(lastMessageHandler, PreStopSignal{})
	}

	for _, child := range ctx.children {
		child.context.commands <- &terminateCommand{}
	}
	ctx.childrenWaitGroup.Wait()

	if ctx.deliverSignals {
		ctx.deliver(lastMessageHandler, PostStopSignal{})
	}
	ctx.parent.childrenWaitGroup.Done()
	if ctx.parent.self != nil {
		ctx.parent.commands <- &childTerminatedCommand{ref: ctx.self}
	}

	for _, listener := range ctx.listeners {
		listener.ref.Tell(ctx, listener.msg)
	}
}

func (ctx *localActorContext) setup(handler SetupHandler) MessageHandler {
	defer func() {
		if err := recover(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "actor setup panic: %s\n", err)
		}
	}()
	return handler(ctx)
}

func (ctx *localActorContext) deliver(messageHandler MessageHandler, msg interface{}) (newHandler MessageHandler) {
	newHandler = Stopped()
	defer func() {
		if err := recover(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "actor panic: %s\n", err)
		}
	}()
	newHandler = messageHandler(msg)
	return newHandler
}

func (ctx *localActorContext) onListenCommand(command *listenCommand) {
	ctx.listeners = append(ctx.listeners, terminateListener{ref: command.ref, msg: command.msg})
}

func (ctx *localActorContext) onChildTerminatedCommand(command *childTerminatedCommand) {
	for i, ref := range ctx.children {
		if ref == command.ref {
			ctx.children = append(ctx.children[:i], ctx.children[i+1:]...)
			return
		}
	}
	panic(fmt.Sprintf("bad child: %+v %+v", ctx.children, command))
}

func (system *actorSystemImpl) Root() ActorRef {
	return system.root
}

func (system *actorSystemImpl) start(root SetupHandler) {
	system.context = newContext(system, nil, nil)
	system.root = system.context.spawn(root)
}
