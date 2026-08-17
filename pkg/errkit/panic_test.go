package errkit_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/pkg/errkit"
)

var _ = Describe("PanicError", func() {
	It("handles string value", func() {
		err := errkit.NewPanicError("test panic string")

		Expect(err.Error()).To(Equal("panic recovered: test panic string"))
		Expect(err.Stack).NotTo(BeNil())
		Expect(string(err.Stack)).To(ContainSubstring("errkit_test"))

		unwrapped := err.Unwrap()
		Expect(unwrapped).To(BeNil())
	})

	It("handles error value", func() {
		innerErr := errors.New("test panic error")
		err := errkit.NewPanicError(innerErr)

		Expect(err.Error()).To(Equal("panic recovered: test panic error"))
		Expect(err.Stack).NotTo(BeNil())

		unwrapped := err.Unwrap()
		Expect(unwrapped).To(Equal(innerErr))
	})
})
