package application

import (
	"log/slog"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/config"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "app", &def)
}

func newCreateInvite(i do.Injector) (*CreateInvite, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	invites, err := do.InvokeAs[InviteSaver](i)
	if err != nil {
		return nil, err
	}

	counter, err := do.InvokeAs[ActiveInviteCounter](i)
	if err != nil {
		return nil, err
	}

	tokens, err := do.InvokeAs[TokenGenerator](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	cfgInvite := cfg.Invite

	return NewCreateInvite(
		log, invites, counter, tokens, ids,
		invite.NewPolicy(cfgInvite.MaxActive), transactor, cfgInvite.TTL,
	), nil
}

type RootCreateInvite struct {
	*CreateInvite
}

func newRootCreateInvite(i do.Injector) (*RootCreateInvite, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	invites, err := do.InvokeAs[InviteSaver](i)
	if err != nil {
		return nil, err
	}

	counter, err := do.InvokeAs[ActiveInviteCounter](i)
	if err != nil {
		return nil, err
	}

	tokens, err := do.InvokeAs[TokenGenerator](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	cfgInvite := cfg.Invite

	ci := NewCreateInvite(
		log, invites, counter, tokens, ids,
		invite.NewUnlimitedPolicy(), transactor, cfgInvite.RootTTL,
	)
	return &RootCreateInvite{CreateInvite: ci}, nil
}

func newBeginInviteRegistration(i do.Injector) (*BeginInviteRegistration, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	invites, err := do.InvokeAs[InviteFinderByCode](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	registrator, err := do.InvokeAs[Registrator](i)
	if err != nil {
		return nil, err
	}

	hash, err := do.InvokeAs[HashGenerator](i)
	if err != nil {
		return nil, err
	}

	return NewBeginInviteRegistration(log, invites, ids, registrator, hash), nil
}

func newSessionIssuer(i do.Injector) (*SessionIssuer, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	sessions, err := do.InvokeAs[RefreshTokenSaver](i)
	if err != nil {
		return nil, err
	}

	refreshTokens, err := do.InvokeAs[TokenGenerator](i)
	if err != nil {
		return nil, err
	}

	accessTokens, err := do.InvokeAs[TokenIssuer](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[UserFinderByID](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}

	cfgSession := cfg.Session

	return NewSessionIssuer(
		log,
		sessions,
		refreshTokens,
		accessTokens,
		ids,
		users,
		transactor,
		cfgSession.RefreshTTL,
		cfgSession.AccessTTL,
	), nil
}

func newFinishInviteRegistration(i do.Injector) (*FinishInviteRegistration, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	invitesFinder, err := do.InvokeAs[InviteFinderByID](i)
	if err != nil {
		return nil, err
	}

	inviteSaver, err := do.InvokeAs[InviteSaver](i)
	if err != nil {
		return nil, err
	}

	userFinder, err := do.InvokeAs[UserFinderByID](i)
	if err != nil {
		return nil, err
	}

	userSaver, err := do.InvokeAs[UserSaver](i)
	if err != nil {
		return nil, err
	}

	credentials, err := do.InvokeAs[CredentialSaver](i)
	if err != nil {
		return nil, err
	}

	keys, err := do.InvokeAs[WrappedKeySaver](i)
	if err != nil {
		return nil, err
	}

	sessionIssuer, err := do.Invoke[*SessionIssuer](i)
	if err != nil {
		return nil, err
	}

	registrator, err := do.InvokeAs[Registrator](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	cfgInvite := cfg.Invite

	return NewFinishInviteRegistration(
		log,
		invitesFinder,
		inviteSaver,
		userFinder,
		userSaver,
		credentials,
		keys,
		sessionIssuer,
		registrator,
		ids,
		transactor,
		key.NewPolicy(cfgInvite.MaxWrappedKey),
	), nil
}

func newBeginLogin(i do.Injector) (*BeginLogin, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	authenticator, err := do.InvokeAs[Authenticator](i)
	if err != nil {
		return nil, err
	}

	return NewBeginLogin(log, authenticator), nil
}

func newRevokeAllUserSessions(i do.Injector) (*RevokeAllUserSessions, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	sessions, err := do.InvokeAs[ActiveSessionsFinder](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[RefreshTokenSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	return NewRevokeAllUserSessions(log, sessions, saver, transactor), nil
}

func newRevokeSession(i do.Injector) (*RevokeSession, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	sessions, err := do.InvokeAs[RefreshTokenFinderByID](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[RefreshTokenSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	return NewRevokeSession(log, sessions, saver, transactor), nil
}

func newLogin(i do.Injector) (*Login, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	credentials, err := do.InvokeAs[CredentialFinder](i)
	if err != nil {
		return nil, err
	}

	credentialSaver, err := do.InvokeAs[CredentialSaver](i)
	if err != nil {
		return nil, err
	}

	wrappedKeys, err := do.InvokeAs[CredentialWrappedKeysFinder](i)
	if err != nil {
		return nil, err
	}

	authenticator, err := do.InvokeAs[Authenticator](i)
	if err != nil {
		return nil, err
	}

	sessionIssuer, err := do.Invoke[*SessionIssuer](i)
	if err != nil {
		return nil, err
	}

	revokeUser, err := do.Invoke[*RevokeAllUserSessions](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	return NewLogin(
		log,
		credentials,
		credentialSaver,
		wrappedKeys,
		authenticator,
		sessionIssuer,
		revokeUser,
		transactor,
	), nil
}

func newRefreshAccessToken(i do.Injector) (*RefreshAccessToken, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	sessions, err := do.InvokeAs[RefreshTokenFinder](i)
	if err != nil {
		return nil, err
	}

	revokeUser, err := do.Invoke[*RevokeAllUserSessions](i)
	if err != nil {
		return nil, err
	}

	sessionIssuer, err := do.Invoke[*SessionIssuer](i)
	if err != nil {
		return nil, err
	}

	hasher, err := do.InvokeAs[HashGenerator](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[Transactor](i)
	if err != nil {
		return nil, err
	}

	return NewRefreshAccessToken(
		log,
		sessions,
		revokeUser,
		sessionIssuer,
		hasher,
		transactor,
	), nil
}

func newRevokeCompromisedChain(i do.Injector) (*RevokeCompromisedChain, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	descendants, err := do.InvokeAs[UserDescendantsFinder](i)
	if err != nil {
		return nil, err
	}

	revokeUser, err := do.Invoke[*RevokeAllUserSessions](i)
	if err != nil {
		return nil, err
	}

	return NewRevokeCompromisedChain(log, descendants, revokeUser), nil
}

func newBootstrapRoot(i do.Injector) (*BootstrapRoot, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[UserSaver](i)
	if err != nil {
		return nil, err
	}

	finder, err := do.InvokeAs[RootFinder](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[IDGenerator](i)
	if err != nil {
		return nil, err
	}

	return NewBootstrapRoot(log, users, finder, ids), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newCreateInvite),
	do.Lazy(newRootCreateInvite),
	do.Lazy(newBeginInviteRegistration),
	do.Lazy(newFinishInviteRegistration),
	do.Lazy(newSessionIssuer),
	do.Lazy(newBeginLogin),
	do.Lazy(newLogin),
	do.Lazy(newRefreshAccessToken),
	do.Lazy(newRevokeAllUserSessions),
	do.Lazy(newRevokeSession),
	do.Lazy(newRevokeCompromisedChain),
	do.Lazy(newBootstrapRoot),
)
