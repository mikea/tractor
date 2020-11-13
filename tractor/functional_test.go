package tractor_test

import (
	"fmt"
	"github.com/mikea/tractor/tractor"
	. "github.com/onsi/ginkgo"
	//. "github.com/onsi/gomega"
)

var _ = Describe("ActorSystem", func() {
	It("supports stopped actor", func() {
		system := tractor.Run(tractor.Stopped())
		system.Wait()
	})

	It("supports actor stopped after setup", func() {
		actor := func(ctx tractor.ActorContext) tractor.Behavior {
			return tractor.Stopped()
		}
		system := tractor.Start(actor)
		system.Wait()
	})

	It("complicatedTest", func() {
		printBehavior := func(ctx tractor.ActorContext) tractor.Behavior {
			return tractor.Receive(func(msg interface{}) tractor.Behavior {
				fmt.Println(msg)
				return tractor.Stopped()
			})
		}

		logBehavior := func(ctx tractor.ActorContext) tractor.Behavior {
			return tractor.Receive(func(msg interface{}) tractor.Behavior {
				child := ctx.Spawn(printBehavior)
				child.Tell(msg)
				if msg == "QUIT" {
					return tractor.Stopped()
				}
				return tractor.Same()
			})
		}

		system := tractor.Start(logBehavior)
		system.Root().Tell("1")
		system.Root().Tell("2")
		system.Root().Tell("3")
		system.Root().Tell("QUIT")

		system.Wait()
	})
})
