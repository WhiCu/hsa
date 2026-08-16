package idgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIdgen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Idgen Suite")
}
