package tractor

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("behavior", func() {
	Describe("isStopped", func() {
		It("returns true for stopped handler", func() {
			Expect(isStopped(Stopped())).To(BeTrue())
		})
	})
})
