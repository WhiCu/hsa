package application_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
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
		registrator = mocks.NewRegistrator(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		si = application.NewSessionIssuer(
			logger.NewNOPSlog(),
			sessions,
			refreshTokens,
			accessTokens,
			ids,
			24*time.Hour,
			15*time.Minute,
		)

		uc = application.NewFinishInviteRegistration(
			logger.NewNOPSlog(),
			invites,
			users,
			credentials,
			keys,
			si,
			registrator,
			ids,
			transactor,
			24*time.Hour,
			15*time.Minute,
		)

		// Мок транзакции: исполняет переданную замыкающую функцию
		transactor.EXPECT().
			RunInTransaction(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Maybe()
	})

	// --- Success Scenarios ---

	Describe("Success Scenarios", func() {
		It("Basic Case: Valid challenge, creates entities, preserves InitialSignCount, issues tokens", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")

				invID := uuid.New()
				userID := uuid.New()
				credExternalID := []byte("external-id")
				pubKey := []byte("pub-key")
				transports := []string{testTransportUSB}
				initialSignCount := uint32(42)

				regResult := application.RegistrationResult{
					UserID:           userID,
					InviteID:         invID,
					CredentialID:     credExternalID,
					PublicKey:        pubKey,
					Transports:       transports,
					InitialSignCount: initialSignCount,
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
					return c.ID() == newCredID &&
						c.UserID() == userID &&
						c.SignCount() == initialSignCount
				})).Return(nil).Once()

				keyID := uuid.New()
				ids.EXPECT().NewID().Return(keyID).Once()

				keys.EXPECT().SaveAll(ctx, mock.MatchedBy(func(ks []*key.WrappedKey) bool {
					return len(ks) == 1 && ks[0].ID() == keyID
				})).Return(nil).Once()

				expectedRefreshCode := testRefreshCode
				refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()

				sessionID := uuid.New()
				ids.EXPECT().NewID().Return(sessionID).Once()
				sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				expectedAccessCode := testAccessCode
				accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return(expectedAccessCode, nil).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
					WrappedKeys: []application.WrappedKeyInput{
						{Scope: key.ScopeMain, WrappedDEK: []byte("dek"), WrapAlgorithm: "alg"},
					},
					DeviceInfo: "device",
					IPAddress:  "192.168.1.100",
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).ToNot(HaveOccurred())
				Expect(out).ToNot(BeNil())
				Expect(out.AccessToken).To(Equal(expectedAccessCode))
				Expect(out.RefreshToken).To(Equal(expectedRefreshCode))
			})
		})

		It("Succeeds with empty WrappedKeys slice", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")
				invID := uuid.New()
				userID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:           userID,
					InviteID:         invID,
					CredentialID:     []byte("ext"),
					PublicKey:        []byte("pub"),
					Transports:       []string{"usb"},
					InitialSignCount: 0,
				}
				registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
				invites.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()
				invites.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
				users.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				ids.EXPECT().NewID().Return(uuid.New()).Once() // Credential ID
				credentials.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				keys.EXPECT().SaveAll(ctx, mock.MatchedBy(func(ks []*key.WrappedKey) bool {
					return len(ks) == 0
				})).Return(nil).Once()

				expectedRefreshCode := "refresh-code"
				refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()
				ids.EXPECT().NewID().Return(uuid.New()).Once() // Session ID
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
	})

	// --- Pre-Transaction & Domain Failures ---

	Describe("Pre-Transaction & Domain Failures", func() {
		It("Fails early when WebAuthn registration finish fails (e.g. forged/expired challenge)", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
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
		})

		It("Returns ErrInviteNotFound when invite is not found in database", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")
				invID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:   uuid.New(),
					InviteID: invID,
				}
				registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				invites.EXPECT().FindByID(ctx, invID).Return(nil, domain.ErrNotFound).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(application.ErrInviteNotFound))
				Expect(out).To(BeNil())
			})
		})
	})

	// --- Transaction Failures & Rollback Checks ---

	Describe("Transaction Failures & Rollback Checks", func() {
		It("Rolls back when invite redemption fails (e.g. invite already used or expired)", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := "challenge-token"
				regResponse := []byte("reg-response")
				invID := uuid.New()
				userID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:   userID,
					InviteID: invID,
				}
				registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				// Уже использованный инвайт
				inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
				_ = inv.Redeem(uuid.New(), time.Now())

				invites.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(invite.ErrAlreadyUsed))
				Expect(out).To(BeNil())
			})
		})

		It("Rolls back when WrappedKeySaver fails during transaction", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
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
						{Scope: key.ScopeMain, WrappedDEK: []byte("dek"), WrapAlgorithm: "alg"},
					},
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(expectedErr))
				Expect(out).To(BeNil())
			})
		})
	})
	Context("FinishInviteRegistration Helper Functions", func() {
		It("WrappedKeyInput String", func() {
			wki := application.WrappedKeyInput{
				Scope:         key.ScopeMain,
				WrappedDEK:    []byte("my-secret-dek"),
				WrapAlgorithm: "AES-256-GCM",
			}

			str := wki.String()
			expected := "WrappedKeyInput{Scope: 0, WrappedDEK: ***REDACTED***, WrapAlgorithm: AES-256-GCM}"
			Expect(str).To(Equal(expected))
		})
	})
})
