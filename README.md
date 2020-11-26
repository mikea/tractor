![build](https://github.com/mikea/tractor/workflows/Go/badge.svg)

# Tractor

*[WORK IN PROGRESS]*

Actor System for Golang.

The goal of the project at this stage is to design an idiomatic go api for actor programming model together with common
services provided by an actor system.

## Documentation

### Defining Actors

#### Setup Handler

To define an actor the setup handler needs to be provided:

```go
type SetupHandler func(ctx ActorContext) MessageHandler
```

This callback is executed by the system every time new instance of the actor is created. The job of the callback is to
initialize all resources, send all initial requests, setup and return the message handler responsible for processing
actor mailbox messages.

#### Message Handler

Message handler callback processes the incoming message and returns a new handler to be used after the message.

```go
type MessageHandler func(message interface{}) MessageHandler
``` 

Two special values are recognized:

- `nil` - signals to use the *same* handler that was used to process the message
- `Stopped()` - signals to the system to terminate the current actor.

#### Setup Parameters

To parametrize an actor create a closure binding initialization parameters:

```go
func Countdown(start int) SetupHandler {
	return func(ctx ActorContext) MessageHandler {
		count := start
		return func(msg interface{}) MessageHandler {
			count = count - 1
			if count == 0 {
				return Stopped()
			}
			return nil
		}
	}
}
```

### Spawning Actors

Any actor can spawn a child by invoking `ctx.Spawn()` method with a setup handler:

```go
ctx.Spawn(Countdown(10))
``` 

### Actor Lifecycle

#### Signals

Receive messages related to the actor lifecycle by enabling signal delivery in the context:

```go
func(ctx ActorContext) MessageHandler {
    ctx.DeliverSignals(true)
    return func(msg interface{}) MessageHandler {
        switch msg.(type) {
        case PostInitSignal:
            // first message delivered after the initialization
        case PreStopSignal:
        	// delivered before terminating children
        case PostStopSignal:
            // delivered after terminating all children
        }
        return nil
    }
}
```

#### Watching Actors

Any actor can be notified when any other actor is terminated:

```go
func(ctx ActorContext) MessageHandler {
    ctx.Watch(ctx.Spawn(child))
    return func(msg interface{}) MessageHandler {
        switch msg.(type) {
        case Terminated:
        	// child was terminated
        }
        return nil
    }
}
```

or with a custom message:

```go
ctx.WatchWith(ctx.Spawn(child), "childTerminated")
```

### Actor System

#### Starting System

To start the system you neeed to provide setup handler for the root actor:

```go
system := tractor.Start(Root())
```

#### Shutting Down

The system shuts down when root actor stops. You can wait for actor system to finish:

```go
system.Wait()
```

#### Communicating With System

Communication with the running system is done through the root actor reference:

```go
system.Root().Tell("request")
```