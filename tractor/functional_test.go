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
		It("All Signals", func() {
			postInitDelivered := false
			preStopDelivered := false
			postStopDelivered := false

			setup := func(ctx ActorContext) MessageHandler {
				ctx.DeliverSignals(true)

				return func(msg interface{}) MessageHandler {
					switch msg.(type) {
					case PostInitSignal:
						Expect(postInitDelivered).To(BeFalse())
						Expect(preStopDelivered).To(BeFalse())
						postInitDelivered = true
						return Stopped()
					case PreStopSignal:
						Expect(postInitDelivered).To(BeTrue())
						Expect(preStopDelivered).To(BeFalse())
						Expect(postStopDelivered).To(BeFalse())
						preStopDelivered = true
						return nil
					case PostStopSignal:
						Expect(postInitDelivered).To(BeTrue())
						Expect(preStopDelivered).To(BeTrue())
						Expect(postStopDelivered).To(BeFalse())
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
			Expect(preStopDelivered).To(BeTrue())
			Expect(postStopDelivered).To(BeTrue())
		})

		Context("PostInit", func() {
			It("Is delivered before any message", func() {
				postInitDelivered := false
				msgDelivered := false

				setup := func(ctx ActorContext) MessageHandler {
					ctx.Self().Tell("test")
					ctx.DeliverSignals(true)

					return func(msg interface{}) MessageHandler {
						switch msg.(type) {
						case PostInitSignal:
							Expect(postInitDelivered).To(BeFalse())
							Expect(msgDelivered).To(BeFalse())
							postInitDelivered = true
						case string:
							Expect(postInitDelivered).To(BeTrue())
							Expect(msgDelivered).To(BeFalse())
							msgDelivered = true
							return Stopped()
						}
						return nil
					}
				}

				system := Start(setup)
				system.Wait()
				Expect(postInitDelivered).To(BeTrue())
				Expect(msgDelivered).To(BeTrue())
			})
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

	Context("ActorContext", func() {
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

		Context("Children", func() {
			It("empty on start", func() {
				actor := func(ctx ActorContext) MessageHandler {
					Expect(ctx.Children()).To(BeEmpty())
					return Stopped()
				}
				system := Start(actor)
				system.Wait()
			})

			It("spawn adds children", func() {
				child := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						return Stopped()
					}
				}

				actor := func(ctx ActorContext) MessageHandler {
					Expect(ctx.Children()).To(BeEmpty())
					ref := ctx.Spawn(child)
					Expect(ctx.Children()).To(HaveLen(1))
					ref.Tell("stop")
					return Stopped()
				}
				system := Start(actor)
				system.Wait()
			})
		})

		Context("Watch", func() {
			It("gets notified of termination", func() {
				child := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						if msg == "stop" {
							return Stopped()
						}
						panic(msg)
					}
				}

				root := func(ctx ActorContext) MessageHandler {
					ref := ctx.Spawn(child)
					ctx.Watch(ref)
					return func(msg interface{}) MessageHandler {
						if msg == "stopChild" {
							ref.Tell("stop")
							return nil
						}
						if _, ok := msg.(Terminated); ok {
							return Stopped()
						}
						panic(msg)
					}
				}

				system := Start(root)
				system.Root().Tell("stopChild")
				system.Wait()
			})
		})
	})

	Context("Children", func() {
		It("children are stopped when parent is", func() {
			childStopped := false
			child := func(ctx ActorContext) MessageHandler {
				ctx.DeliverSignals(true)
				return func(msg interface{}) MessageHandler {
					if _, ok := msg.(PostStopSignal); ok {
						Expect(childStopped).To(BeFalse())
						childStopped = true
					}
					return nil
				}
			}

			parent := func(ctx ActorContext) MessageHandler {
				ctx.Spawn(child)
				return Stopped()
			}

			system := Start(parent)
			system.Wait()
			Expect(childStopped).To(BeTrue())
		})
	})
})

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
