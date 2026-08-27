package application_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

const (
	testChallengeToken = "challenge-token"
	testRefreshCode    = "refresh-code"
	testRefreshHash    = "refresh-hash"
	testAccessCode     = "access-code"
	testTransportUSB   = "usb"
	testString         = "test"
)

var testT *testing.T

func TestApplication(t *testing.T) {
	testT = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "Application Suite")
}

type Mocks struct {
	InviteSaver                 *mocks.InviteSaver
	ActiveInviteCounter         *mocks.ActiveInviteCounter
	TokenGenerator              *mocks.TokenGenerator
	IDGenerator                 *mocks.IDGenerator
	InviteFinderByCode          *mocks.InviteFinderByCode
	InviteFinderByID            *mocks.InviteFinderByID
	Registrator                 *mocks.Registrator
	HashGenerator               *mocks.HashGenerator
	RefreshTokenSaver           *mocks.RefreshTokenSaver
	TokenIssuer                 *mocks.TokenIssuer
	UserFinderByID              *mocks.UserFinderByID
	UserSaver                   *mocks.UserSaver
	CredentialSaver             *mocks.CredentialSaver
	WrappedKeySaver             *mocks.WrappedKeySaver
	Authenticator               *mocks.Authenticator
	ActiveSessionsFinder        *mocks.ActiveSessionsFinder
	RefreshTokenFinderByID      *mocks.RefreshTokenFinderByID
	CredentialFinder            *mocks.CredentialFinder
	CredentialWrappedKeysFinder *mocks.CredentialWrappedKeysFinder
	RefreshTokenFinder          *mocks.RefreshTokenFinder
	UserDescendantsFinder       *mocks.UserDescendantsFinder
	RootFinder                  *mocks.RootFinder
	Transactor                  *mocks.Transactor
}

var defaultCfg = application.Config{
	Invite: application.InviteConfig{
		MaxActive: 3,
		TTL:       72 * time.Hour,
		RootTTL:   (7 * 24) * time.Hour,

		MaxWrappedKey: 10,
	},
	Session: application.SessionConfig{
		RefreshTTL: 30 * 24 * time.Hour,
		AccessTTL:  15 * time.Minute,
	},
}

func MockPackage(i do.Injector, t interface {
	mock.TestingT
	Cleanup(func())
}) *Mocks {
	m := &Mocks{
		InviteSaver:                 mocks.NewInviteSaver(t),
		ActiveInviteCounter:         mocks.NewActiveInviteCounter(t),
		TokenGenerator:              mocks.NewTokenGenerator(t),
		IDGenerator:                 mocks.NewIDGenerator(t),
		InviteFinderByCode:          mocks.NewInviteFinderByCode(t),
		InviteFinderByID:            mocks.NewInviteFinderByID(t),
		Registrator:                 mocks.NewRegistrator(t),
		HashGenerator:               mocks.NewHashGenerator(t),
		RefreshTokenSaver:           mocks.NewRefreshTokenSaver(t),
		TokenIssuer:                 mocks.NewTokenIssuer(t),
		UserFinderByID:              mocks.NewUserFinderByID(t),
		UserSaver:                   mocks.NewUserSaver(t),
		CredentialSaver:             mocks.NewCredentialSaver(t),
		WrappedKeySaver:             mocks.NewWrappedKeySaver(t),
		Authenticator:               mocks.NewAuthenticator(t),
		ActiveSessionsFinder:        mocks.NewActiveSessionsFinder(t),
		RefreshTokenFinderByID:      mocks.NewRefreshTokenFinderByID(t),
		CredentialFinder:            mocks.NewCredentialFinder(t),
		CredentialWrappedKeysFinder: mocks.NewCredentialWrappedKeysFinder(t),
		RefreshTokenFinder:          mocks.NewRefreshTokenFinder(t),
		UserDescendantsFinder:       mocks.NewUserDescendantsFinder(t),
		RootFinder:                  mocks.NewRootFinder(t),
		Transactor:                  mocks.NewTransactor(t),
	}

	do.OverrideValue[*slog.Logger](i, logger.NewNOPSlog())
	do.OverrideValue[application.Config](i, defaultCfg)

	m.Transactor.
		EXPECT().
		RunInTransaction(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Maybe()
	do.OverrideValue[application.Transactor](i, m.Transactor)

	// Переопределяем интерфейсы моками
	do.OverrideValue[application.InviteSaver](i, m.InviteSaver)
	do.OverrideValue[application.ActiveInviteCounter](i, m.ActiveInviteCounter)
	do.OverrideValue[application.TokenGenerator](i, m.TokenGenerator)
	do.OverrideValue[application.IDGenerator](i, m.IDGenerator)
	do.OverrideValue[application.InviteFinderByCode](i, m.InviteFinderByCode)
	do.OverrideValue[application.InviteFinderByID](i, m.InviteFinderByID)
	do.OverrideValue[application.Registrator](i, m.Registrator)
	do.OverrideValue[application.HashGenerator](i, m.HashGenerator)
	do.OverrideValue[application.RefreshTokenSaver](i, m.RefreshTokenSaver)
	do.OverrideValue[application.TokenIssuer](i, m.TokenIssuer)
	do.OverrideValue[application.UserFinderByID](i, m.UserFinderByID)
	do.OverrideValue[application.UserSaver](i, m.UserSaver)
	do.OverrideValue[application.CredentialSaver](i, m.CredentialSaver)
	do.OverrideValue[application.WrappedKeySaver](i, m.WrappedKeySaver)
	do.OverrideValue[application.Authenticator](i, m.Authenticator)
	do.OverrideValue[application.ActiveSessionsFinder](i, m.ActiveSessionsFinder)
	do.OverrideValue[application.RefreshTokenFinderByID](i, m.RefreshTokenFinderByID)
	do.OverrideValue[application.CredentialFinder](i, m.CredentialFinder)
	do.OverrideValue[application.CredentialWrappedKeysFinder](i, m.CredentialWrappedKeysFinder)
	do.OverrideValue[application.RefreshTokenFinder](i, m.RefreshTokenFinder)
	do.OverrideValue[application.UserDescendantsFinder](i, m.UserDescendantsFinder)
	do.OverrideValue[application.RootFinder](i, m.RootFinder)

	return m
}
