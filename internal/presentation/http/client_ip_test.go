package http_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	internalHTTP "github.com/whicu/hsa/internal/presentation/http"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("ClientIPMiddleware", func() {
	It("extracts direct RemoteAddr when no trusted proxies configured", func() {
		mw, err := internalHTTP.NewClientIPMiddleware(logger.NewNOPSlog())
		Expect(err).NotTo(HaveOccurred())

		var capturedIP netip.Addr
		handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			var ok bool
			capturedIP, ok = internalHTTP.ClientIPFromContext(r.Context())
			Expect(ok).To(BeTrue())
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.195:45678"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")

		handler.ServeHTTP(httptest.NewRecorder(), req)
		Expect(capturedIP).To(Equal(netip.MustParseAddr("203.0.113.195")))
	})

	It("extracts client IP from X-Forwarded-For through trusted proxies chain", func() {
		mw, err := internalHTTP.NewClientIPMiddleware(logger.NewNOPSlog(), "10.0.0.0/8", "192.168.0.0/16")
		Expect(err).NotTo(HaveOccurred())

		var capturedIP netip.Addr
		handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			var ok bool
			capturedIP, ok = internalHTTP.ClientIPFromContext(r.Context())
			Expect(ok).To(BeTrue())
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.50, 192.168.1.10")

		handler.ServeHTTP(httptest.NewRecorder(), req)
		Expect(capturedIP).To(Equal(netip.MustParseAddr("203.0.113.50")))
	})

	It("returns error when CIDR format is invalid", func() {
		_, err := internalHTTP.NewClientIPMiddleware(logger.NewNOPSlog(), "invalid-cidr-mask")
		Expect(err).To(HaveOccurred())
	})
})
