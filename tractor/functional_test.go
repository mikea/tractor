package tractor

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ActorSystem", func() {
	It("supports actor stopped after setup", func() {
		initialized := false
		actor := func(ctx ActorContext) MessageHandler {
			initialized = true
			return Stopped()
		}
		system := Start(actor)
		system.Wait()

		Expect(initialized).To(BeTrue())
	})

	It("countdown", func() {
		system := Start(Countdown(10))
		for i := 0; i < 10; i++ {
			system.Root().Tell("")
		}
		system.Wait()
	})

	Context("Signals", func() {
		It("PostInit/PostStop", func() {
			postInitDelivered := false
			postStopDelivered := false

			setup := func(ctx ActorContext) MessageHandler {
				ctx.DeliverSignals(true)

				return func(msg interface{}) MessageHandler {
					switch msg.(type) {
					case PostInitSignal:
						Expect(postStopDelivered).To(BeFalse())
						Expect(postInitDelivered).To(BeFalse())
						postInitDelivered = true
						return Stopped()
					case PostStopSignal:
						Expect(postStopDelivered).To(BeFalse())
						Expect(postInitDelivered).To(BeTrue())
						postStopDelivered = true
						return nil
					default:
						panic(fmt.Sprintf("unsupported message %T", msg))
					}
				}
			}

			system := Start(setup)
			system.Wait()

			Expect(postInitDelivered).To(BeTrue())
			Expect(postStopDelivered).To(BeTrue())
		})
	})

	Context("Spawn", func() {
		It("Spawn start a child", func() {
			child := func(ctx ActorContext) MessageHandler {
				ctx.Parent().Tell("stop")
				return Stopped()
			}

			parent := func(ctx ActorContext) MessageHandler {
				ctx.Spawn(child)
				return func(msg interface{}) MessageHandler {
					if msg == "stop" {
						return Stopped()
					}
					panic(msg)
				}
			}

			system := Start(parent)
			system.Wait()
		})
	})

	Context("Self", func() {
		It("Tell to self", func() {
			startReceived := false
			stopReceived := false

			parent := func(ctx ActorContext) MessageHandler {
				ctx.Self().Tell("start")
				ctx.Self().Tell("stop")
				ctx.Self().Tell("wrong")
				return func(msg interface{}) MessageHandler {
					if msg == "start" {
						Expect(startReceived).To(BeFalse())
						Expect(stopReceived).To(BeFalse())
						startReceived = true
						return nil
					}
					if msg == "stop" {
						Expect(startReceived).To(BeTrue())
						Expect(stopReceived).To(BeFalse())
						stopReceived = true
						return Stopped()
					}
					panic(msg)
				}
			}

			system := Start(parent)
			system.Wait()
			Expect(startReceived).To(BeTrue())
			Expect(stopReceived).To(BeTrue())
		})
	})

	Context("Behavior", func() {
		It("Switching behavior", func() {
			stopBehavior := func(msg interface{}) MessageHandler {
				if msg == "stop" {
					return Stopped()
				}
				panic(msg)
			}
			startBehavior := func(msg interface{}) MessageHandler {
				if msg == "start" {
					return stopBehavior
				}
				panic(msg)
			}

			system := Start(func(ctx ActorContext) MessageHandler {
				return startBehavior
			})
			system.Root().Tell("start")
			system.Root().Tell("stop")
			system.Wait()
		})
	})
})

func Countdown(start int) SetupHandler {
	count := start
	return func(ctx ActorContext) MessageHandler {
		return func(msg interface{}) MessageHandler {
			count = count - 1
			if count == 0 {
				return Stopped()
			}
			return nil
		}
	}
}
