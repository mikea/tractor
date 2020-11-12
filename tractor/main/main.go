package main

import (
	"fmt"
	"github.com/mikea/tractor/tractor"
)

func main() {
	system := tractor.Start(PrintActor())
	system.Root().Tell("1")
	system.Root().Tell("2")
	system.Root().Tell("3")
	system.Root().Tell("QUIT")

	system.Wait()
}

func PrintActor() tractor.Behavior {
	return tractor.Handle(func(msg interface{}) tractor.Behavior {
		if msg == "QUIT" {
			return tractor.Stopped()
		}
		fmt.Println(msg)
		return tractor.Same()
	})
}
