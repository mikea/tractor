package tractor_test

import (
	"fmt"
	"github.com/mikea/tractor/tractor"
	. "github.com/onsi/ginkgo"
	//. "github.com/onsi/gomega"
)

var _ = Describe("Foo", func() {
	It("fails", func() {
		system := tractor.Start(LogActor)
		system.Root().Tell("1")
		system.Root().Tell("2")
		system.Root().Tell("3")
		system.Root().Tell("QUIT")

		system.Wait()
	})
})

func LogActor(ctx tractor.ActorContext) tractor.Behavior {
	return tractor.Receive(func(msg interface{}) tractor.Behavior {
		child := ctx.Spawn(PrintActor)
		child.Tell(msg)
		if msg == "QUIT" {
			return tractor.Stopped()
		}
		return tractor.Same()
	})
}

func PrintActor(_ tractor.ActorContext) tractor.Behavior {
	return tractor.Receive(func(msg interface{}) tractor.Behavior {
		fmt.Println(msg)
		return tractor.Stopped()
	})
}
