package errkit_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/pkg/errkit"
)

type customError1Error struct{ msg string }

func (e customError1Error) Error() string { return e.msg }

type customError2Error struct{ msg string }

func (e customError2Error) Error() string { return e.msg }

func TestErrMapPkg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ErrMap Suite")
}

var _ = Describe("ErrMap", func() {
	It("resolves errors with and without defaults", func() {
		errMap := &errkit.Registry[int]{}

		err1 := customError1Error{"error 1"}
		err2 := customError2Error{"error 2"}
		err3 := errors.New("regular error")

		errkit.Register[customError1Error, int](errMap, func(_ error) int {
			return 100
		})
		errkit.Register[customError2Error, int](errMap, func(_ error) int {
			return 200
		})
		errkit.RegisterDefault(errMap, func(_ error) int {
			return 500
		})

		Expect(errMap.Resolve(err1)).To(Equal(100))
		Expect(errMap.Resolve(err2)).To(Equal(200))

		wrappedErr := errkit.Append(err1, errors.New("extra context"))
		Expect(errMap.Resolve(wrappedErr)).To(Equal(100))

		Expect(errMap.Resolve(err3)).To(Equal(500))
		Expect(errMap.Resolve(nil)).To(Equal(500))
	})

	It("handles missing default safely", func() {
		errMap := &errkit.Registry[int]{}
		errkit.Register[customError1Error, int](errMap, func(_ error) int {
			return 100
		})
		err2 := customError2Error{"error 2"}

		Expect(errMap.Resolve(err2)).To(Equal(0))
	})
})
