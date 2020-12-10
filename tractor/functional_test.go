package tractor

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ActorSystem", func() {
	Context("Countdown Example", func() {
		It("stops after correct number of messages", func() {
			system := Start(Countdown(10))
			for i := 0; i < 10; i++ {
				system.Root().Tell(system.Context(), "")
			}
			system.Wait()
		})
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
					ctx.Self().Tell(ctx, "test")
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

	Context("SetupHandler", func() {
		It("stopping the actor", func() {
			initialized := false
			actor := func(ctx ActorContext) MessageHandler {
				initialized = true
				return Stopped()
			}
			system := Start(actor)
			system.Wait()

			Expect(initialized).To(BeTrue())
		})

		It("nil stops the actor too", func() {
			initialized := false
			actor := func(ctx ActorContext) MessageHandler {
				initialized = true
				return nil
			}
			system := Start(actor)
			system.Wait()

			Expect(initialized).To(BeTrue())
		})

		It("panic stops the actor", func() {
			initialized := false
			actor := func(ctx ActorContext) MessageHandler {
				initialized = true
				panic("stop")
			}
			system := Start(actor)
			system.Wait()

			Expect(initialized).To(BeTrue())
		})
	})

	Context("MessageHandler", func() {
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
			system.Root().Tell(system.Context(), "start")
			system.Root().Tell(system.Context(), "stop")
			system.Wait()
		})

		It("Panic stops the actor", func() {
			child := func(ctx ActorContext) MessageHandler {
				return func(msg interface{}) MessageHandler {
					panic(msg)
				}
			}

			actor := func(ctx ActorContext) MessageHandler {
				Expect(ctx.Children()).To(BeEmpty())
				ref := ctx.Spawn(child)
				Expect(ctx.Children()).To(HaveLen(1))
				ctx.Watch(ref)
				ref.Tell(ctx, "stop")

				return func(msg interface{}) MessageHandler {
					if _, ok := msg.(Terminated); ok {
						Expect(ctx.Children()).To(HaveLen(0))
						return Stopped()
					}
					panic(msg)
				}
			}

			system := Start(actor)
			system.Wait()
		})
	})

	Context("ActorContext", func() {
		Context("Self", func() {
			It("Tell to self", func() {
				startReceived := false
				stopReceived := false

				parent := func(ctx ActorContext) MessageHandler {
					ctx.Self().Tell(ctx, "start")
					ctx.Self().Tell(ctx, "stop")
					ctx.Self().Tell(ctx, "wrong")
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
					ctx.Parent().Tell(ctx, "stop")
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
					ref.Tell(ctx, "stop")
					return Stopped()
				}
				system := Start(actor)
				system.Wait()
			})

			It("stop removes children", func() {
				child := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						return Stopped()
					}
				}

				actor := func(ctx ActorContext) MessageHandler {
					Expect(ctx.Children()).To(BeEmpty())
					ref := ctx.Spawn(child)
					Expect(ctx.Children()).To(HaveLen(1))
					ctx.Watch(ref)
					ref.Tell(ctx, "stop")

					return func(msg interface{}) MessageHandler {
						if _, ok := msg.(Terminated); ok {
							Expect(ctx.Children()).To(HaveLen(0))
							return Stopped()
						}
						panic(msg)
					}
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
							ref.Tell(ctx, "stop")
							return nil
						}
						if _, ok := msg.(Terminated); ok {
							return Stopped()
						}
						panic(msg)
					}
				}

				system := Start(root)
				system.Root().Tell(system.Context(), "stopChild")
				system.Wait()
			})

			It("can send custom message", func() {
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
					ctx.WatchWith(ref, "childTerminated")
					return func(msg interface{}) MessageHandler {
						if msg == "stopChild" {
							ref.Tell(ctx, "stop")
							return nil
						} else if msg == "childTerminated" {
							return Stopped()
						}
						panic(msg)
					}
				}

				system := Start(root)
				system.Root().Tell(system.Context(), "stopChild")
				system.Wait()
			})
		})

		Context("Sender", func() {
			It("works", func() {
				child := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						if msg == "ping" {
							ctx.Sender().Tell(ctx, "pong")
						}
						return nil
					}
				}

				replyRecieved := false
				parent := func(ctx ActorContext) MessageHandler {
					ref := ctx.Spawn(child)
					ref.Tell(ctx, "ping")

					return func(msg interface{}) MessageHandler {
						if msg == "pong" {
							replyRecieved = true
							return Stopped()
						}
						panic(msg)
					}
				}

				system := Start(parent)
				system.Wait()
				Expect(replyRecieved).To(BeTrue())
			})
		})

		Context("Ask", func() {
			It("works", func() {
				child := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						if msg == "ping" {
							ctx.Sender().Tell(ctx, "pong")
						}
						return nil
					}
				}

				replyRecieved := false
				parent := func(ctx ActorContext) MessageHandler {
					ref := ctx.Spawn(child)

					return func(msg interface{}) MessageHandler {
						if msg == "ask" {
							reply := <-ctx.Ask(ref, "ping")
							if reply == "pong" {
								replyRecieved = true
								return Stopped()
							}
							panic(reply)
						}
						panic(msg)
					}
				}

				system := Start(parent)
				system.Root().Tell(system.Context(), "ask")
				system.Wait()
				Expect(replyRecieved).To(BeTrue())
			})

			It("system ask", func() {
				setupRoot := func(ctx ActorContext) MessageHandler {
					return func(msg interface{}) MessageHandler {
						if msg == "ping" {
							ctx.Sender().Tell(ctx, "pong")
							return Stopped()
						}
						panic(msg)
					}
				}

				system := Start(setupRoot)
				reply := <-system.Context().Ask(system.Root(), "ping")
				system.Wait()
				Expect(reply).To(Equal("pong"))
			})
		})

		Context("Stash", func() {
			It("stash/unstash all work", func() {

				acc := ""
				add := func(msg interface{}) MessageHandler {
					if msg == "stop" {
						return Stopped()
					} else if s, ok := msg.(string); ok {
						acc = acc + s
						return nil
					}

					panic(msg)
				}

				rootSetup := func(ctx ActorContext) MessageHandler {
					stash := ctx.NewStash(1000)
					return func(msg interface{}) MessageHandler {
						if msg == "unstash" {
							return stash.UnstashAll(add)
						} else {
							stash.Stash(msg)
						}

						return nil
					}
				}

				system := Start(rootSetup)
				system.Root().Tell(system.Context(), "1")
				system.Root().Tell(system.Context(), "2")
				system.Root().Tell(system.Context(), "unstash")
				system.Root().Tell(system.Context(), "stop")
				system.Wait()
				Expect(acc).To(Equal("12"))
			})
		})
	})

	Context("Children", func() {
		It("children are stopped when parent is", func() {
			childStopped := false
			childSetup := func(ctx ActorContext) MessageHandler {
				ctx.DeliverSignals(true)
				return func(msg interface{}) MessageHandler {
					if _, ok := msg.(PostStopSignal); ok {
						Expect(childStopped).To(BeFalse())
						childStopped = true
					}
					return nil
				}
			}

			parentSetup := func(ctx ActorContext) MessageHandler {
				ctx.Spawn(childSetup)
				return Stopped()
			}

			system := Start(parentSetup)
			system.Wait()
			Expect(childStopped).To(BeTrue())
		})
	})

	Context("Counter", func() {
		Context("typed ref", func() {
			It("Works", func() {
				rootSetup := func(ctx ActorContext) MessageHandler {
					counter := CounterRef{ctx.Spawn(Counter())}

					zero := <-counter.GetAndIncrement(ctx)
					Expect(zero).To(Equal(0))

					one := <-counter.GetAndIncrement(ctx)
					Expect(one).To(Equal(1))

					return Stopped()
				}

				system := Start(rootSetup)
				system.Wait()
			})
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

type getAndIncrement struct{}

func Counter() SetupHandler {
	return func(ctx ActorContext) MessageHandler {
		count := 0
		return func(m interface{}) MessageHandler {
			switch m.(type) {
			case getAndIncrement:
				ctx.Sender().Tell(ctx, count)
				count++
			}
			return nil
		}
	}
}

type CounterRef struct {
	Ref ActorRef
}

func (ref CounterRef) GetAndIncrement(ctx ActorContext) chan interface{} {
	return ctx.Ask(ref.Ref, getAndIncrement{})
}
