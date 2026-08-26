package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	api "github.com/whicu/hsa/api/http"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
	internalHTTP "github.com/whicu/hsa/internal/presentation/http"
)

const (
	fieldRawID                = "rawId"
	fieldChallengeToken       = "challengeToken"
	fieldAuthResponse         = "authenticationResponse"
	fieldRegistrationResponse = "registrationResponse"
	fieldWrappedKeys          = "wrappedKeys"
	fieldInviteCode           = "inviteCode"
	valChallenge123           = "challenge-123"
	valWrapAlgorithm          = "xchacha20poly1305-prf-kek"
	valRegCredID              = "reg-cred-id"
)

type remoteAddrBinder struct {
	handler http.Handler
}

func (b *remoteAddrBinder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.RemoteAddr == "" {
		req.RemoteAddr = "127.0.0.1:12345"
	}
	rec := httptest.NewRecorder()
	b.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

var _ = Describe("Handler", func() {
	var (
		injector   do.Injector
		mocks      *internalHTTP.Mocks
		router     http.Handler
		e          *httpexpect.Expect
		testUserID uuid.UUID
		authHeader string
	)

	BeforeEach(func() {
		injector = do.New(internalHTTP.Package)
		mocks = internalHTTP.MockPackage(injector, GinkgoT())

		var testCfg internalHTTP.Config
		var err error
		testCfg, err = do.Invoke[internalHTTP.Config](injector)
		Expect(err).NotTo(HaveOccurred())

		testCfg.TrustedProxies = []string{"127.0.0.1/32"}
		testCfg.CORS.AllowedOrigins = []string{"*"}

		do.OverrideValue(injector, testCfg)

		router, err = do.Invoke[http.Handler](injector)
		Expect(err).NotTo(HaveOccurred())

		e = httpexpect.WithConfig(httpexpect.Config{
			Client: &http.Client{
				Transport: &remoteAddrBinder{handler: router},
			},
			Reporter: httpexpect.NewRequireReporter(GinkgoT()),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(GinkgoT(), true),
			},
		})

		testUserID = uuid.New()
		authHeader = "Bearer valid-access-token"
	})

	mockAuthSuccess := func() {
		mocks.SecurityHandler.EXPECT().
			HandleBearerAuth(
				mock.Anything,
				mock.Anything,
				mock.MatchedBy(func(b api.BearerAuth) bool {
					return b.Token == "valid-access-token"
				}),
			).
			RunAndReturn(func(
				ctx context.Context,
				_ api.OperationName,
				_ api.BearerAuth,
			) (context.Context, error) {
				return internalHTTP.WithUserID(ctx, testUserID), nil
			}).
			Maybe()
	}

	mockAuthUnauthorized := func() {
		mocks.SecurityHandler.EXPECT().
			HandleBearerAuth(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, _ api.OperationName, _ api.BearerAuth) (context.Context, error) {
				return ctx, internalHTTP.ErrUnauthenticated
			}).
			Maybe()
	}

	// =========================================================================
	// GET /auth/verify
	// =========================================================================
	Describe("GET /auth/verify", func() {
		It("returns 204 No Content with X-Auth-User header for authenticated user", func() {
			mockAuthSuccess()

			e.GET("/auth/verify").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent).
				Header("X-Auth-User").IsEqual(testUserID.String())
		})

		It("returns 401 Unauthorized when token is invalid or missing", func() {
			mockAuthUnauthorized()

			resp := e.GET("/auth/verify").
				WithHeader("Authorization", "Bearer expired-or-invalid").
				Expect().
				Status(http.StatusUnauthorized)

			resp.Header("WWW-Authenticate").IsEqual("Bearer")

			body := resp.JSON().Object()
			body.Value("error").Object().Value("code").IsEqual("UNAUTHORIZED")
		})
	})

	// =========================================================================
	// POST /invites
	// =========================================================================
	Describe("POST /invites", func() {
		It("returns 201 Created with code and expiresAt on success", func() {
			mockAuthSuccess()
			expiresAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

			mocks.CreateInvite.EXPECT().
				Execute(mock.Anything, testUserID).
				Return("invite-token-abc-123", expiresAt, nil).
				Once()

			resp := e.POST("/invites").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusCreated).
				JSON().Object()

			resp.Value("code").IsEqual("invite-token-abc-123")
			resp.Value("expiresAt").String().AsDateTime().IsEqual(expiresAt)
		})

		It("returns 409 Conflict when user active invite limit is exceeded", func() {
			mockAuthSuccess()

			mocks.CreateInvite.EXPECT().
				Execute(mock.Anything, testUserID).
				Return("", time.Time{}, invite.ErrTooManyActive).
				Once()

			resp := e.POST("/invites").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusConflict).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("INVITE_LIMIT_EXCEEDED")
		})
	})

	// =========================================================================
	// POST /login/begin
	// =========================================================================
	Describe("POST /login/begin", func() {
		It("returns 200 OK with challenge token and request options", func() {
			opts := map[string]any{
				"publicKey": map[string]any{
					"challenge": "test-challenge",
				},
			}
			optsBytes, err := json.Marshal(opts)
			Expect(err).NotTo(HaveOccurred())

			mocks.BeginLogin.EXPECT().
				Execute(mock.Anything).
				Return("login-challenge-token", optsBytes, nil).
				Once()

			resp := e.POST("/login/begin").
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			resp.Value(fieldChallengeToken).IsEqual("login-challenge-token")
			resp.Value("requestOptions").IsEqual(opts)
		})
	})

	// =========================================================================
	// POST /login
	// =========================================================================
	Describe("POST /login", func() {
		It("returns 200 OK with TokenPair and WrappedKeys on successful WebAuthn authentication", func() {
			authResp := map[string]any{
				"id":             "credential-id",
				fieldRawID:       "raw-credential-id",
				"clientDataJSON": "eyJ0ZXN0IjoxfQ==",
			}
			authRespBytes, err := json.Marshal(authResp)
			Expect(err).NotTo(HaveOccurred())

			reqBody := map[string]any{
				fieldChallengeToken: valChallenge123,
				fieldAuthResponse:   authResp,
				"deviceInfo":        "Safari on iOS",
			}

			mocks.FinishLogin.EXPECT().
				Execute(mock.Anything, mock.MatchedBy(func(in application.LoginInput) bool {
					return in.ChallengeToken == valChallenge123 &&
						string(in.AuthenticationResponse) == string(authRespBytes) &&
						in.DeviceInfo == "Safari on iOS" &&
						in.IPAddress == netip.MustParseAddr("198.51.100.1")
				})).
				Return(&application.LoginOutput{
					AccessToken:  "access-token-123",
					RefreshToken: "refresh-token-456",
					WrappedKeys: []application.WrappedKeyOutput{
						{
							Scope:         key.ScopeMain,
							WrappedDEK:    []byte("wrapped-dek-payload"),
							WrapAlgorithm: valWrapAlgorithm,
						},
					},
				}, nil).
				Once()

			resp := e.POST("/login").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(reqBody).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			tokenPair := resp.Value("tokenPair").Object()
			tokenPair.Value("accessToken").IsEqual("access-token-123")
			tokenPair.Value("refreshToken").IsEqual("refresh-token-456")

			keys := resp.Value(fieldWrappedKeys).Array()
			keys.Length().IsEqual(1)
			keys.Value(0).Object().Value("scope").IsEqual("main")
			keys.Value(0).Object().Value("wrapAlgorithm").IsEqual(valWrapAlgorithm)
		})

		It("returns 401 Unauthorized when credential clone is detected", func() {
			authResp := map[string]any{"id": "cloned-id"}

			mocks.FinishLogin.EXPECT().
				Execute(mock.Anything, mock.Anything).
				Return(nil, application.ErrCredentialCloneSuspected).
				Once()

			resp := e.POST("/login").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(map[string]any{
					fieldChallengeToken: "c-token",
					fieldAuthResponse:   authResp,
				}).
				Expect().
				Status(http.StatusUnauthorized).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("CREDENTIAL_CLONE_SUSPECTED")
		})

		It("returns 401 Unauthorized when credential is not registered", func() {
			authResp := map[string]any{"id": "unknown-id"}

			mocks.FinishLogin.EXPECT().
				Execute(mock.Anything, mock.Anything).
				Return(nil, application.ErrCredentialNotFound).
				Once()

			resp := e.POST("/login").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(map[string]any{
					fieldChallengeToken: "c-token",
					fieldAuthResponse:   authResp,
				}).
				Expect().
				Status(http.StatusUnauthorized).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("CREDENTIAL_NOT_FOUND")
		})
	})

	// =========================================================================
	// POST /registration/begin
	// =========================================================================
	Describe("POST /registration/begin", func() {
		It("returns 200 OK with challenge token and creation options", func() {
			creationOpts := map[string]any{
				"publicKey": map[string]any{
					"rp": map[string]any{
						"name": "HSA",
					},
				},
			}
			optsBytes, err := json.Marshal(creationOpts)
			Expect(err).NotTo(HaveOccurred())

			mocks.BeginInviteRegistration.EXPECT().
				Execute(mock.Anything, "valid-invite-code").
				Return("reg-challenge-token", optsBytes, nil).
				Once()

			resp := e.POST("/registration/begin").
				WithJSON(map[string]any{
					fieldInviteCode: "valid-invite-code",
				}).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			resp.Value(fieldChallengeToken).IsEqual("reg-challenge-token")
			resp.Value("creationOptions").IsEqual(creationOpts)
		})

		It("returns 404 Not Found when invite does not exist", func() {
			mocks.BeginInviteRegistration.EXPECT().
				Execute(mock.Anything, "missing-code").
				Return("", nil, domain.ErrNotFound).
				Once()

			resp := e.POST("/registration/begin").
				WithJSON(map[string]any{
					fieldInviteCode: "missing-code",
				}).
				Expect().
				Status(http.StatusNotFound).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("INVITE_NOT_FOUND")
		})

		It("returns 409 Conflict when invite is expired", func() {
			mocks.BeginInviteRegistration.EXPECT().
				Execute(mock.Anything, "expired-code").
				Return("", nil, invite.ErrExpired).
				Once()

			resp := e.POST("/registration/begin").
				WithJSON(map[string]any{
					fieldInviteCode: "expired-code",
				}).
				Expect().
				Status(http.StatusConflict).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("INVITE_EXPIRED")
		})

		It("returns 409 Conflict when invite is already used", func() {
			mocks.BeginInviteRegistration.EXPECT().
				Execute(mock.Anything, "used-code").
				Return("", nil, invite.ErrAlreadyUsed).
				Once()

			resp := e.POST("/registration/begin").
				WithJSON(map[string]any{
					fieldInviteCode: "used-code",
				}).
				Expect().
				Status(http.StatusConflict).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("INVITE_ALREADY_USED")
		})
	})

	// =========================================================================
	// POST /registration/complete
	// =========================================================================
	Describe("POST /registration/complete", func() {
		It("returns 201 Created with TokenPair upon successful registration", func() {
			regResp := map[string]any{
				"id":       valRegCredID,
				fieldRawID: "raw-id-data",
				"type":     "public-key",
			}
			regRespBytes, err := json.Marshal(regResp)
			Expect(err).NotTo(HaveOccurred())

			reqBody := map[string]any{
				fieldChallengeToken:       "reg-challenge-123",
				fieldRegistrationResponse: regResp,
				"deviceInfo":              "MacBook Chrome",
				fieldWrappedKeys: []map[string]any{
					{
						"scope":         "main",
						"wrappedDek":    "d3JhcHBlZC1kZWstYnl0ZXM=", // Base64: "wrapped-dek-bytes"
						"wrapAlgorithm": valWrapAlgorithm,
					},
				},
			}

			mocks.FinishInviteRegistration.EXPECT().
				Execute(mock.Anything, mock.MatchedBy(func(in application.FinishInviteRegistrationInput) bool {
					return in.ChallengeToken == "reg-challenge-123" &&
						string(in.RegistrationResponse) == string(regRespBytes) &&
						in.DeviceInfo == "MacBook Chrome" &&
						in.IPAddress == netip.MustParseAddr("198.51.100.1") &&
						len(in.WrappedKeys) == 1 &&
						in.WrappedKeys[0].Scope == key.ScopeMain
				})).
				Return(&application.FinishInviteRegistrationOutput{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
				}, nil).
				Once()

			resp := e.POST("/registration/complete").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(reqBody).
				Expect().
				Status(http.StatusCreated).
				JSON().Object()

			resp.Value("accessToken").IsEqual("new-access-token")
			resp.Value("refreshToken").IsEqual("new-refresh-token")
		})

		It("returns 400 Bad Request when wrapped keys are missing in usecase", func() {
			regResp := map[string]any{
				"id":       valRegCredID,
				fieldRawID: "raw-id-data",
				"type":     "public-key",
			}

			mocks.FinishInviteRegistration.EXPECT().
				Execute(mock.Anything, mock.Anything).
				Return(nil, application.ErrWrappedKeysRequired).
				Once()

			resp := e.POST("/registration/complete").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(map[string]any{
					fieldChallengeToken:       valChallenge123,
					fieldRegistrationResponse: regResp,
					fieldWrappedKeys: []map[string]any{
						{
							"scope":         "main",
							"wrappedDek":    "d3JhcHBlZC1kZWstYnl0ZXM=",
							"wrapAlgorithm": valWrapAlgorithm,
						},
					},
				}).
				Expect().
				Status(http.StatusBadRequest).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("WRAPPED_KEYS_REQUIRED")
		})

		It("returns 400 Bad Request on OpenAPI schema validation error when wrappedKeys is empty", func() {
			regResp := map[string]any{"id": valRegCredID}

			resp := e.POST("/registration/complete").
				WithHeader("X-Forwarded-For", "198.51.100.1").
				WithJSON(map[string]any{
					fieldChallengeToken:       "challenge",
					fieldRegistrationResponse: regResp,
					fieldWrappedKeys:          []any{},
				}).
				Expect().
				Status(http.StatusBadRequest).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("VALIDATION_ERROR")
		})
	})

	// =========================================================================
	// POST /token/refresh
	// =========================================================================
	Describe("POST /token/refresh", func() {
		It("returns 200 OK with rotated TokenPair on valid refresh token", func() {
			mocks.RefreshAccessToken.EXPECT().
				Execute(mock.Anything, "raw-refresh-token").
				Return("new-access", "new-refresh", nil).
				Once()

			resp := e.POST("/token/refresh").
				WithJSON(map[string]any{
					"refreshToken": "raw-refresh-token",
				}).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			resp.Value("accessToken").IsEqual("new-access")
			resp.Value("refreshToken").IsEqual("new-refresh")
		})

		It("returns 401 Unauthorized when refresh token reuse is detected", func() {
			mocks.RefreshAccessToken.EXPECT().
				Execute(mock.Anything, "compromised-refresh-token").
				Return("", "", application.ErrRefreshTokenReuseDetected).
				Once()

			resp := e.POST("/token/refresh").
				WithJSON(map[string]any{
					"refreshToken": "compromised-refresh-token",
				}).
				Expect().
				Status(http.StatusUnauthorized).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("REFRESH_TOKEN_REUSE_DETECTED")
		})
	})

	// =========================================================================
	// DELETE /sessions/{sessionId}
	// =========================================================================
	Describe("DELETE /sessions/{sessionId}", func() {
		It("returns 204 No Content when session is revoked by owner", func() {
			mockAuthSuccess()
			targetSessionID := uuid.New()

			mocks.RevokeSession.EXPECT().
				Execute(mock.Anything, targetSessionID, testUserID).
				Return(nil).
				Once()

			e.DELETE("/sessions/{sessionId}", targetSessionID.String()).
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent)
		})

		It("returns 403 Forbidden when trying to revoke another user's session", func() {
			mockAuthSuccess()
			targetSessionID := uuid.New()

			mocks.RevokeSession.EXPECT().
				Execute(mock.Anything, targetSessionID, testUserID).
				Return(application.ErrSessionForbidden).
				Once()

			resp := e.DELETE("/sessions/{sessionId}", targetSessionID.String()).
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusForbidden).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("SESSION_FORBIDDEN")
		})

		It("returns 404 Not Found when session does not exist", func() {
			mockAuthSuccess()
			targetSessionID := uuid.New()

			mocks.RevokeSession.EXPECT().
				Execute(mock.Anything, targetSessionID, testUserID).
				Return(application.ErrSessionNotFound).
				Once()

			resp := e.DELETE("/sessions/{sessionId}", targetSessionID.String()).
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNotFound).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("SESSION_NOT_FOUND")
		})
	})

	// =========================================================================
	// POST /admin/users/{userId}/revoke-chain
	// =========================================================================
	Describe("POST /admin/users/{userId}/revoke-chain", func() {
		It("returns 204 No Content on successful chain revocation", func() {
			mockAuthSuccess()
			targetUserID := uuid.New()

			mocks.RevokeCompromisedChain.EXPECT().
				Execute(mock.Anything, targetUserID).
				Return(nil).
				Once()

			e.POST("/admin/users/{userId}/revoke-chain", targetUserID.String()).
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent)
		})

		It("returns 500 Internal Server Error when usecase fails", func() {
			mockAuthSuccess()
			targetUserID := uuid.New()

			mocks.RevokeCompromisedChain.EXPECT().
				Execute(mock.Anything, targetUserID).
				Return(errors.New("db failure")).
				Once()

			resp := e.POST("/admin/users/{userId}/revoke-chain", targetUserID.String()).
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusInternalServerError).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("INTERNAL_ERROR")
		})

		It("returns 401 Unauthorized when accessing admin route without token", func() {
			mockAuthUnauthorized()
			targetUserID := uuid.New()

			resp := e.POST("/admin/users/{userId}/revoke-chain", targetUserID.String()).
				Expect().
				Status(http.StatusUnauthorized).
				JSON().Object()

			resp.Value("error").Object().Value("code").IsEqual("UNAUTHORIZED")
		})

		It("returns 400 Bad Request when userId is not a valid UUID", func() {
			mockAuthSuccess()

			e.POST("/admin/users/{userId}/revoke-chain", "invalid-uuid").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusBadRequest).
				JSON().Object().Value("error").Object().Value("code").IsEqual("VALIDATION_ERROR")
		})
	})

	// =========================================================================
	// CORS Middleware
	// =========================================================================
	Describe("CORS Middleware", func() {
		It("handles preflight OPTIONS requests with CORS headers", func() {
			e.OPTIONS("/login").
				WithHeader("Origin", "http://localhost:3000").
				WithHeader("Access-Control-Request-Method", "POST").
				Expect().
				Status(http.StatusOK).
				Header("Access-Control-Allow-Origin").IsEqual("*")
		})
		It("should handle preflight OPTIONS requests and return correct CORS headers", func() {
			res := e.OPTIONS("/login/begin").
				WithHeader("Origin", "http://localhost:3000").
				WithHeader("Access-Control-Request-Method", "POST").
				WithHeader("Access-Control-Request-Headers", "Authorization, Content-Type").
				Expect().
				// go-chi/cors по умолчанию возвращает 204 No Content для preflight
				Status(http.StatusOK)
				// Проверяем разрешенный источник
			res.Header("Access-Control-Allow-Origin").IsEqual("*")
			// Проверяем разрешенные методы (используем Contains, так как go-chi объединяет их через запятую)
			res.Header("Access-Control-Allow-Methods").Contains("POST")
			// Проверяем разрешенные заголовки
			res.Header("Access-Control-Allow-Headers").Contains("Authorization")
			res.Header("Access-Control-Allow-Headers").Contains("Content-Type")
			// Проверяем Max-Age
			res.Header("Access-Control-Max-Age").IsEqual("300")
		})

		It("should attach CORS headers to actual requests", func() {
			mockAuthSuccess()
			// Вызываем реальный эндпоинт с Origin
			e.GET("/auth/verify").
				WithHeader("Origin", "http://localhost:3000").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent).
				Header("Access-Control-Allow-Origin").IsEqual("*")
		})

		It("should expose requested headers on actual requests", func() {
			// Для проверки ExposedHeaders лучше всего подходит /auth/verify,
			// который возвращает X-Auth-User
			mockAuthSuccess() // Включаем успешную авторизацию для этого теста

			res := e.GET("/auth/verify").
				WithHeader("Origin", "http://localhost:3000").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent)
			res.Header("Access-Control-Allow-Origin").IsEqual("*")
			res.Header("Access-Control-Expose-Headers").Contains("X-Auth-User")
		})
		It("should expose requested headers on actual requests", func() {
			mockAuthSuccess()

			res := e.GET("/auth/verify").
				WithHeader("Origin", "http://localhost:3000").
				WithHeader("Authorization", authHeader).
				Expect().
				Status(http.StatusNoContent)

			res.Header("Access-Control-Allow-Origin").IsEqual("*")
			res.Header("Access-Control-Expose-Headers").Contains("X-Auth-User")
		})
	})
})
