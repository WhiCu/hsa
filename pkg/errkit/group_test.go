package errkit_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/pkg/errkit"
)

var _ = Describe("Group", func() {
	It("executes successfully", func() {
		var g errkit.Group

		var count int
		g.Go(func() error {
			count++
			return nil
		})

		err := g.Wait()
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1))

		g.Add(nil) // should do nothing
		Expect(g.Wait()).To(Succeed())
	})

	It("collects errors", func() {
		var g errkit.Group

		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		g.Go(func() error {
			return err1
		})
		g.Go(func() error {
			return err2
		})

		err := g.Wait()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error 1"))
		Expect(err.Error()).To(ContainSubstring("error 2"))
	})

	It("calls finally block", func() {
		var g errkit.Group

		finallyCalled := false
		g.Finally(func() error {
			finallyCalled = true
			return nil
		})

		g.Wait()
		Expect(finallyCalled).To(BeTrue())
	})

	It("returns error from finally block", func() {
		var g errkit.Group

		err1 := errors.New("finally error")

		g.Finally(func() error {
			return err1
		})

		err := g.Wait()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("finally error"))
	})

	It("allows direct error addition", func() {
		var g errkit.Group

		err1 := errors.New("direct error")
		g.Add(err1)

		err := g.Wait()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("direct error"))
	})

	It("safely recovers from panics and converts them to PanicError", func() {
		var g errkit.Group

		g.Go(func() error {
			panic("test panic")
		})

		err := g.Wait()
		Expect(err).To(HaveOccurred())

		var panicErr *errkit.PanicError
		Expect(errors.As(err, &panicErr)).To(BeTrue())
		Expect(panicErr.Value).To(Equal("test panic"))
		Expect(err.Error()).To(ContainSubstring("panic recovered: test panic"))
	})
})
