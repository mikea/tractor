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

	It("complicatedTest", func() {
		printBehavior := func(ctx ActorContext) MessageHandler {
			return func(msg interface{}) MessageHandler {
				fmt.Println(msg)
				return Stopped()
			}
		}

		logBehavior := func(ctx ActorContext) MessageHandler {
			return func(msg interface{}) MessageHandler {
				child := ctx.Spawn(printBehavior)
				child.Tell(msg)
				if msg == "QUIT" {
					return Stopped()
				}
				return nil
			}
		}

		system := Start(logBehavior)
		system.Root().Tell("1")
		system.Root().Tell("2")
		system.Root().Tell("3")
		system.Root().Tell("QUIT")

		system.Wait()
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
