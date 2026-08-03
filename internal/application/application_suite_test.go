package application_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testChallengeToken = "challenge-token"
	testRefreshCode    = "refresh-code"
	testRefreshHash    = "refresh-hash"
	testAccessCode     = "access-code"
	testTransportUSB   = "usb"
)

var testT *testing.T

func TestApplication(t *testing.T) {
	testT = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "Application Suite")
}
