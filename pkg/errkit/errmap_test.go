package errkit_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/pkg/errkit"
)

func TestErrKit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ErrKit Suite")
}

type customError1 struct{ msg string }

func (e customError1) Error() string { return e.msg }

type customError2 struct{ code int }

func (e customError2) Error() string { return fmt.Sprintf("code: %d", e.code) }

var errSentinel = errors.New("sentinel error")

var _ = Describe("Registry", func() {
	Describe("OnAs", func() {
		It("matches by error type, including wrapped errors", func() {
			r := errkit.New[int]().
				OnAs(func(e customError1) int { return 100 }).
				OnAs(func(e customError2) int { return e.code })

			// Прямое совпадение
			val, ok := r.Resolve(customError1{msg: "err1"})
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(100))

			val, ok = r.Resolve(customError2{code: 204})
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(204))

			// Обернутая ошибка (%w)
			wrapped := fmt.Errorf("wrapped: %w", customError1{msg: "nested"})
			val, ok = r.Resolve(wrapped)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(100))
		})
	})

	Describe("OnIs", func() {
		It("matches target sentinel error, including wrapped errors", func() {
			r := errkit.New[int]().
				OnIs(func(_ error) int { return 404 }, errSentinel)

			val, ok := r.Resolve(errSentinel)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(404))

			wrapped := fmt.Errorf("context: %w", errSentinel)
			val, ok = r.Resolve(wrapped)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(404))

			// Несовпадающая ошибка
			val, ok = r.Resolve(errors.New("other error"))
			Expect(ok).To(BeFalse())
			Expect(val).To(Equal(0))
		})
	})

	Describe("OnMatch", func() {
		It("matches when predicate returns true", func() {
			r := errkit.New[int]().
				OnMatch(
					func(err error) bool {
						return strings.Contains(err.Error(), "connection refused")
					},
					func(_ error) int {
						return 503
					},
				)

			val, ok := r.Resolve(errors.New("dial tcp 127.0.0.1: connection refused"))
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(503))

			val, ok = r.Resolve(errors.New("timeout"))
			Expect(ok).To(BeFalse())
			Expect(val).To(Equal(0))
		})
	})

	Describe("Default fallback", func() {
		It("uses default handler when no other handlers match", func() {
			r := errkit.New[int]().
				OnAs(func(_ customError1) int {
					return 100
				}).
				Default(func(_ error) int {
					return 500
				})

			// Совпадение по OnAs
			val, ok := r.Resolve(customError1{})
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(100))

			// Не совпало -> отрабатывает Default
			val, ok = r.Resolve(errors.New("unregistered error"))
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(500))
		})

		It("returns zero value and false when no handlers match and no default is set", func() {
			r := errkit.New[int]().
				OnAs(func(_ customError1) int {
					return 100
				})

			val, ok := r.Resolve(errors.New("unregistered error"))
			Expect(ok).To(BeFalse())
			Expect(val).To(Equal(0))
		})
	})

	Describe("Order of evaluation", func() {
		It("executes the first matching handler in order of declaration", func() {
			r := errkit.New[string]().
				OnIs(func(_ error) string {
					return "first"
				}, errSentinel).
				OnMatch(func(_ error) bool {
					return true
				}, func(_ error) string {
					return "second"
				})

			val, ok := r.Resolve(errSentinel)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("first"))
		})
	})

	Describe("Handle", func() {
		It("returns mapped value and nil error on successful resolution", func() {
			r := errkit.New[int]().
				OnIs(func(_ error) int {
					return 400
				}, errSentinel)

			val, err := r.Handle(errSentinel)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal(400))
		})

		It("returns zero value and the original error when unresolved", func() {
			r := errkit.New[int]().
				OnIs(func(_ error) int {
					return 400
				}, errSentinel)

			unhandledErr := errors.New("unhandled error")
			val, err := r.Handle(unhandledErr)

			Expect(err).To(MatchError(unhandledErr))
			Expect(val).To(Equal(0))
		})
	})
})
