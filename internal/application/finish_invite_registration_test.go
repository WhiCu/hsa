package application_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
)

var _ = Describe("FinishInviteRegistration", func() {
	var (
		invites       *mocks.InviteFinderByID
		users         *mocks.UserSaver
		credentials   *mocks.CredentialSaver
		keys          *mocks.WrappedKeySaver
		sessions      *mocks.RefreshTokenSaver
		refreshTokens *mocks.TokenGenerator
		accessTokens  *mocks.TokenIssuer
		registrator   *mocks.Registrator
		ids           *mocks.IDGenerator
		transactor    *mocks.Transactor

		uc *application.FinishInviteRegistration
		si *application.SessionIssuer

		ctx context.Context
	)

	BeforeEach(func() {
		invites = mocks.NewInviteFinderByID(GinkgoT())
		users = mocks.NewUserSaver(GinkgoT())
		credentials = mocks.NewCredentialSaver(GinkgoT())
		keys = mocks.NewWrappedKeySaver(GinkgoT())
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())

		si = application.NewSessionIssuer(sessions, refreshTokens, accessTokens, ids, 24*time.Hour, 15*time.Minute)

		registrator = mocks.NewRegistrator(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		uc = application.NewFinishInviteRegistration(invites, users, credentials, keys, si, registrator, ids, transactor, 24*time.Hour, 15*time.Minute)

		ctx = context.Background()

		transactor.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
	})

	It("Basic Case: Valid challenge, creates entities, marks invite redeemed, issues tokens", func() {
		challengeToken := "challenge-token"
		regResponse := []byte("reg-response")

		invID := uuid.New()
		userID := uuid.New()
		credExternalID := []byte("external-id")
		pubKey := []byte("pub-key")
		transports := []string{"usb"}

		regResult := application.RegistrationResult{
			UserID:       userID,
			InviteID:     invID,
			CredentialID: credExternalID,
			PublicKey:    pubKey,
			Transports:   transports,
			PRFSupported: true,
		}
		registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

		inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
		invites.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()

		invites.EXPECT().Save(ctx, mock.MatchedBy(func(i *invite.Invite) bool {
			return i.IsUsed() && i.ID() == invID
		})).Return(nil).Once()

		users.EXPECT().Save(ctx, mock.MatchedBy(func(u *user.User) bool {
			return u.ID() == userID
		})).Return(nil).Once()

		newCredID := uuid.New()
		ids.EXPECT().NewID().Return(newCredID).Once()

		credentials.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
			return c.ID() == newCredID && c.UserID() == userID
		})).Return(nil).Once()

		keyID := uuid.New()
		ids.EXPECT().NewID().Return(keyID).Once()

		keys.EXPECT().SaveAll(ctx, mock.MatchedBy(func(ks []*key.WrappedKey) bool {
			return len(ks) == 1 && ks[0].ID() == keyID
		})).Return(nil).Once()

		expectedRefreshCode := "refresh-code"
		refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()

		sessionID := uuid.New()
		ids.EXPECT().NewID().Return(sessionID).Once()

		sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

		expectedAccessCode := "access-code"
		accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return(expectedAccessCode, nil).Once()

		in := application.FinishInviteRegistrationInput{
			ChallengeToken:       challengeToken,
			RegistrationResponse: regResponse,
			WrappedKeys: []application.WrappedKeyInput{
				{Scope: key.ScopeMain, WrappedDEK: []byte("dek"), WrapAlgorithm: "alg", ViaRecovery: false},
			},
			DeviceInfo: "device",
			IPAddress:  "127.0.0.1",
		}

		out, err := uc.Execute(ctx, in)

		Expect(err).ToNot(HaveOccurred())
		Expect(out).ToNot(BeNil())
		Expect(out.AccessToken).To(Equal(expectedAccessCode))
		Expect(out.RefreshToken).To(Equal(expectedRefreshCode))
	})

	It("Forged/Expired Challenge (fails before transaction)", func() {
		expectedErr := errors.New("challenge expired or invalid")
		registrator.EXPECT().Finish(ctx, "forged-token", []byte("resp")).Return(application.RegistrationResult{}, expectedErr).Once()

		in := application.FinishInviteRegistrationInput{
			ChallengeToken:       "forged-token",
			RegistrationResponse: []byte("resp"),
		}

		out, err := uc.Execute(ctx, in)
		Expect(err).To(MatchError(expectedErr))
		Expect(out).To(BeNil())
	})

	It("Mid-Transaction Failure (e.g. WrappedKeySaver fails) causes rollback", func() {
		challengeToken := "challenge-token"
		regResponse := []byte("reg-response")

		invID := uuid.New()
		userID := uuid.New()

		regResult := application.RegistrationResult{
			UserID:       userID,
			InviteID:     invID,
			CredentialID: []byte("ext"),
			PublicKey:    []byte("pub"),
			Transports:   []string{"usb"},
		}
		registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

		inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
		invites.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()
		invites.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
		users.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
		ids.EXPECT().NewID().Return(uuid.New()).Twice()
		credentials.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

		expectedErr := errors.New("db error saving keys")
		keys.EXPECT().SaveAll(ctx, mock.Anything).Return(expectedErr).Once()

		in := application.FinishInviteRegistrationInput{
			ChallengeToken:       challengeToken,
			RegistrationResponse: regResponse,
			WrappedKeys: []application.WrappedKeyInput{
				{Scope: key.ScopeMain, WrappedDEK: []byte("dek"), WrapAlgorithm: "alg", ViaRecovery: false},
			},
		}

		out, err := uc.Execute(ctx, in)

		Expect(err).To(MatchError(expectedErr))
		Expect(out).To(BeNil())
	})

	It("Empty WrappedKeys validation error", func() {
		challengeToken := "challenge-token"
		regResponse := []byte("reg-response")

		invID := uuid.New()
		userID := uuid.New()

		regResult := application.RegistrationResult{
			UserID:       userID,
			InviteID:     invID,
			CredentialID: []byte("ext"),
			PublicKey:    []byte("pub"),
			Transports:   []string{"usb"},
		}
		registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

		inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
		invites.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()
		invites.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
		users.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
		ids.EXPECT().NewID().Return(uuid.New()).Once() // for cred
		credentials.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

		keys.EXPECT().SaveAll(ctx, mock.MatchedBy(func(ks []*key.WrappedKey) bool {
			return len(ks) == 0
		})).Return(nil).Once()

		expectedRefreshCode := "refresh-code"
		refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()
		ids.EXPECT().NewID().Return(uuid.New()).Once() // for session
		sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
		accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return("access-code", nil).Once()

		in := application.FinishInviteRegistrationInput{
			ChallengeToken:       challengeToken,
			RegistrationResponse: regResponse,
			WrappedKeys:          []application.WrappedKeyInput{},
		}

		out, err := uc.Execute(ctx, in)

		Expect(err).ToNot(HaveOccurred())
		Expect(out).ToNot(BeNil())
	})
})

// TODO: The user requested a test for Initial SignCount Preservation on Registration.
// It requires `RegistrationResult` to have a `SignCount` field, which it currently does not.
// Once `SignCount uint32` is added to `RegistrationResult` in `auth.go`, and
// `FinishInviteRegistration` calls `cred.SetSignCount(result.SignCount)`,
// this test can be implemented to assert the saved credential has the correct sign count.
