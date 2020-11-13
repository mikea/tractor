package tractor_test

import (
	"github.com/mikea/tractor/tractor"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Behavior", func() {
	Describe("IsStopped", func() {
		It("returns true for stopped handler", func() {
			Expect(tractor.IsStopped(tractor.Stopped())).To(BeTrue())
		})
	})
})
