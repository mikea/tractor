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
	ref.context.mailbox.Tell(envelope{msg: msg, sender: ctx.Self()})
}

type terminateListener struct {
	ref ActorRef
	msg interface{}
}

type envelope struct {
	sender ActorRef
	msg    interface{}
}

type mailbox struct {
	messages chan envelope
	commands chan interface{}
	stash    []envelope
}

func (m *mailbox) takeCommand() interface{} {
	select {
	case cmd := <-m.commands:
		return cmd
	default:
		return nil
	}
}

func (m *mailbox) TellCommand(msg interface{}) {
	m.commands <- msg
}

func (m *mailbox) take() interface{} {
	if len(m.stash) > 0 {
		env := m.stash[0]
		m.stash = m.stash[1:]
		return env
	}
	select {
	case cmd := <-m.commands:
		return cmd
	case env := <-m.messages:
		return env
	}
}

func (m *mailbox) Tell(e envelope) {
	m.messages <- e
}

func (m *mailbox) unstashAll(buffer []envelope) {
	m.stash = append(m.stash, buffer...)
}

type localActorContext struct {
	system            *actorSystemImpl
	parent            *localActorContext
	childrenWaitGroup *sync.WaitGroup
	self              *localActorRef
	deliverSignals    bool
	children          []*localActorRef
	listeners         []terminateListener
	currentEnvelope   *envelope
	mailbox           mailbox
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
	actor.(*localActorRef).context.mailbox.TellCommand(&listenCommand{ref: ctx.self, msg: msg})
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
		mailbox:           newMailbox(),
	}
}

func newMailbox() mailbox {
	return mailbox{
		messages: make(chan envelope, defaultMailboxSize),
		commands: make(chan interface{}, defaultCommandsSize),
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

		msg := ctx.mailbox.take()

		switch command := msg.(type) {
		case envelope:
			ctx.currentEnvelope = &command
			if newHandler := ctx.deliver(messageHandler, command.msg); newHandler != nil {
				messageHandler = newHandler
			}
			ctx.currentEnvelope = nil
		case *terminateCommand:
			messageHandler = Stopped()
		case *listenCommand:
			ctx.onListenCommand(command)
		case *childTerminatedCommand:
			ctx.onChildTerminatedCommand(command)
		default:
			panic(fmt.Sprintf("Bad command: %T", command))
		}
	}

	// drain commands if any
l:
	for {
		cmd := ctx.mailbox.takeCommand()
		if cmd == nil {
			break l
		}

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
	}

	if ctx.deliverSignals {
		ctx.deliver(lastMessageHandler, PreStopSignal{})
	}

	for _, child := range ctx.children {
		child.context.mailbox.TellCommand(&terminateCommand{})
	}
	ctx.childrenWaitGroup.Wait()

	if ctx.deliverSignals {
		ctx.deliver(lastMessageHandler, PostStopSignal{})
	}
	ctx.parent.childrenWaitGroup.Done()
	if ctx.parent.self != nil {
		ctx.parent.mailbox.TellCommand(&childTerminatedCommand{ref: ctx.self})
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

type stashBuffer struct {
	context *localActorContext
	buffer  []envelope
}

func (s *stashBuffer) Stash(msg interface{}) {
	s.buffer = append(s.buffer, envelope{
		sender: s.context.currentEnvelope.sender,
		msg:    msg,
	})
}

func (s *stashBuffer) UnstashAll(handler MessageHandler) MessageHandler {
	s.context.mailbox.unstashAll(s.buffer)
	s.buffer = nil
	return handler
}

func (s *stashBuffer) Unstash(handler MessageHandler, count int) MessageHandler {
	s.context.mailbox.unstashAll(s.buffer[:count])
	s.buffer = s.buffer[count:]
	return handler
}

func (ctx *localActorContext) NewStash(size int) StashBuffer {
	return &stashBuffer{
		context: ctx,
	}
}
