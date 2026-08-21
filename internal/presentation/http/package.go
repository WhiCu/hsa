package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"
	api "github.com/whicu/hsa/api/http"
	"github.com/whicu/hsa/internal/config"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/internal/presentation/http/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "http", &def)
}

func newSecurityHandler(i do.Injector) (api.SecurityHandler, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke logger: %w", err)
	}
	verifier, err := do.InvokeAs[*crypto.AccessTokenVerifier](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke AccessTokenVerifier: %w", err)
	}
	return NewSecurityHandler(logger, verifier), nil
}

func newHandler(i do.Injector) (*Handler, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke logger: %w", err)
	}
	createInvite, err := do.InvokeAs[CreateInvite](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke CreateInvite: %w", err)
	}

	beginLogin, err := do.InvokeAs[BeginLogin](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke BeginLogin: %w", err)
	}

	login, err := do.InvokeAs[FinishLogin](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke FinishLogin: %w", err)
	}

	beginRegistration, err := do.InvokeAs[BeginInviteRegistration](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke BeginInviteRegistration: %w", err)
	}

	finishRegistration, err := do.InvokeAs[FinishInviteRegistration](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke FinishInviteRegistration: %w", err)
	}

	refreshAccessToken, err := do.InvokeAs[RefreshAccessToken](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke RefreshAccessToken: %w", err)
	}

	revokeCompromisedChain, err := do.InvokeAs[RevokeCompromisedChain](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke RevokeCompromisedChain: %w", err)
	}

	revokeSession, err := do.InvokeAs[RevokeSession](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke RevokeSession: %w", err)
	}

	return NewHandler(
		logger,
		createInvite,
		beginLogin,
		login,
		beginRegistration,
		finishRegistration,
		refreshAccessToken,
		revokeCompromisedChain,
		revokeSession,
	), nil
}

func newRouter(i do.Injector) (http.Handler, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke logger: %w", err)
	}
	h, err := do.Invoke[*Handler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke Handler: %w", err)
	}

	secHandler, err := do.InvokeAs[api.SecurityHandler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke SecurityHandler: %w", err)
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke config: %w", err)
	}

	router, err := NewRouter(
		logger,
		h,
		secHandler,
		cfg.RequestSizeLimit,
		cfg.TrustedProxies,
		cfg.CORS.AllowedOrigins,
	)
	if err != nil {
		return nil, fmt.Errorf("http: init router: %w", err)
	}

	return router, nil
}

func newServer(i do.Injector) (*http.Server, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke config: %w", err)
	}

	router, err := do.Invoke[http.Handler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke router: %w", err)
	}

	return NewServer(
		cfg.HostPort(),
		cfg.ReadTimeout,
		cfg.ReadHeaderTimeout,
		cfg.WriteTimeout,
		cfg.IdleTimeout,
		cfg.MaxHeaderBytes,
		router,
	), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newSecurityHandler),
	do.Lazy(newHandler),
	do.Lazy(newRouter),
	do.Lazy(newServer),
)

type Mocks struct {
	CreateInvite             *mocks.CreateInvite
	BeginLogin               *mocks.BeginLogin
	FinishLogin              *mocks.FinishLogin
	BeginInviteRegistration  *mocks.BeginInviteRegistration
	FinishInviteRegistration *mocks.FinishInviteRegistration
	RefreshAccessToken       *mocks.RefreshAccessToken
	RevokeCompromisedChain   *mocks.RevokeCompromisedChain
	RevokeSession            *mocks.RevokeSession
	SecurityHandler          *mocks.SecurityHandler
}

func MockPackage(i do.Injector, t interface {
	mock.TestingT
	Cleanup(func())
}) *Mocks {
	m := &Mocks{
		CreateInvite:             mocks.NewCreateInvite(t),
		BeginLogin:               mocks.NewBeginLogin(t),
		FinishLogin:              mocks.NewFinishLogin(t),
		BeginInviteRegistration:  mocks.NewBeginInviteRegistration(t),
		FinishInviteRegistration: mocks.NewFinishInviteRegistration(t),
		RefreshAccessToken:       mocks.NewRefreshAccessToken(t),
		RevokeCompromisedChain:   mocks.NewRevokeCompromisedChain(t),
		RevokeSession:            mocks.NewRevokeSession(t),
		SecurityHandler:          mocks.NewSecurityHandler(t),
	}

	do.OverrideValue(i, logger.NewNOPSlog())
	do.OverrideValue(i, m.CreateInvite)
	do.OverrideValue(i, m.BeginLogin)
	do.OverrideValue(i, m.FinishLogin)
	do.OverrideValue(i, m.BeginInviteRegistration)
	do.OverrideValue(i, m.FinishInviteRegistration)
	do.OverrideValue(i, m.RefreshAccessToken)
	do.OverrideValue(i, m.RevokeCompromisedChain)
	do.OverrideValue(i, m.RevokeSession)
	do.OverrideValue[api.SecurityHandler](i, m.SecurityHandler)

	do.OverrideValue(i, defaultCfg)

	return m
}
