package application_test

import (
	"errors"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
)

var _ = Describe("FinishInviteRegistration", func() {
	var (
		injector do.Injector
		m        *Mocks
		uc       *application.FinishInviteRegistration
	)

	BeforeEach(func() {
		injector = do.New(application.Package)

		m = MockPackage(injector, GinkgoT())

		var err error
		uc, err = do.Invoke[*application.FinishInviteRegistration](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Success Scenarios", func() {
		It("Basic Case: Valid challenge, creates entities, preserves InitialSignCount, issues tokens", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")

				invID := uuid.New()
				creatorID := uuid.New()
				userID := uuid.New()
				credExternalID := []byte("external-id")
				pubKey := []byte("pub-key")
				transports := []string{testTransportUSB}
				initialSignCount := uint32(42)

				regResult := application.RegistrationResult{
					UserID:           userID,
					InviteID:         invID,
					ExternalID:       credExternalID,
					PublicKey:        pubKey,
					Transports:       transports,
					InitialSignCount: initialSignCount,
				}
				m.Registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				// 1. Погашение инвайта
				inv, _ := invite.New(invID, creatorID, "hash", 24*time.Hour, time.Now())
				m.InviteFinderByID.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()

				m.InviteSaver.EXPECT().Save(ctx, mock.MatchedBy(func(i *invite.Invite) bool {
					return i.IsUsed() && i.ID() == invID
				})).Return(nil).Once()

				// 2. Поиск создателя инвайта
				creatorUser, _ := user.NewRoot(creatorID, time.Now())
				m.UserFinderByID.EXPECT().FindByID(ctx, creatorID).Return(creatorUser, nil).Once()

				// 3. Создание пользователя
				m.UserSaver.EXPECT().Save(ctx, mock.MatchedBy(func(u *user.User) bool {
					// Проверяем, что роль унаследована от создателя (Root создает Admin)
					return u.ID() == userID && u.Role() == user.Admin
				})).Return(nil).Once()

				// 4. Создание креда
				newCredID := uuid.New()
				m.IDGenerator.EXPECT().NewID().Return(newCredID).Once()

				m.CredentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.ID() == newCredID &&
						c.UserID() == userID &&
						c.SignCount() == initialSignCount
				})).Return(nil).Once()

				// 5. Сохранение Wrapped Keys
				keyID := uuid.New()
				m.IDGenerator.EXPECT().NewID().Return(keyID).Once()

				m.WrappedKeySaver.EXPECT().Save(ctx, mock.MatchedBy(func(ks []*key.WrappedKey) bool {
					return len(ks) == 1 && ks[0].ID() == keyID
				})).Return(nil).Once()

				// 6. Выпуск сессии (Refresh Token)
				expectedRefreshCode := testRefreshCode
				m.TokenGenerator.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()

				sessionID := uuid.New()
				m.IDGenerator.EXPECT().NewID().Return(sessionID).Once()
				m.RefreshTokenSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				// 7. Выпуск сессии (Access Token)
				// SessionIssuer читает пользователя из базы, чтобы узнать его роль
				newUser := user.Reconstruct(userID, user.Admin, &creatorID, time.Now())
				m.UserFinderByID.EXPECT().FindByID(ctx, userID).Return(newUser, nil).Once()

				expectedAccessCode := testAccessCode
				m.TokenIssuer.EXPECT().IssueAccessToken(userID, user.Admin, 15*time.Minute).Return(expectedAccessCode, nil).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
					WrappedKeys: []application.WrappedKeyInput{
						{Scope: key.ScopeMain, WrappedDEK: []byte("dek"), WrapAlgorithm: "alg"},
					},
					DeviceInfo: "device",
					IPAddress:  netip.MustParseAddr("192.168.1.100"),
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).ToNot(HaveOccurred())
				Expect(out).ToNot(BeNil())
				Expect(out.AccessToken).To(Equal(expectedAccessCode))
				Expect(out.RefreshToken).To(Equal(expectedRefreshCode))
			})
		})
	})

	// --- Pre-Transaction & Domain Failures ---

	Describe("Pre-Transaction & Domain Failures", func() {
		It("Fails with empty WrappedKeys slice before calling DB", func(ctx SpecContext) {
			in := application.FinishInviteRegistrationInput{
				ChallengeToken:       testChallengeToken,
				RegistrationResponse: []byte("reg-response"),
				WrappedKeys:          []application.WrappedKeyInput{},
			}

			out, err := uc.Execute(ctx, in)

			Expect(err).To(MatchError(application.ErrWrappedKeysRequired))
			Expect(out).To(BeNil())
		})

		It("Fails early when WebAuthn registration finish fails", func(ctx SpecContext) {
			expectedErr := errors.New("challenge expired or invalid")
			m.Registrator.EXPECT().Finish(ctx, "forged-token", []byte("resp")).Return(application.RegistrationResult{}, expectedErr).Once()

			in := application.FinishInviteRegistrationInput{
				ChallengeToken:       "forged-token",
				RegistrationResponse: []byte("resp"),
				WrappedKeys: []application.WrappedKeyInput{
					{Scope: key.ScopeMain, WrappedDEK: []byte{}, WrapAlgorithm: testString},
				},
			}

			out, err := uc.Execute(ctx, in)

			Expect(err).To(MatchError(expectedErr))
			Expect(out).To(BeNil())
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
				m.Registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				m.InviteFinderByID.EXPECT().FindByID(ctx, invID).Return(nil, domain.ErrNotFound).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
					WrappedKeys: []application.WrappedKeyInput{
						{Scope: key.ScopeMain, WrappedDEK: []byte{}, WrapAlgorithm: testString},
					},
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(application.ErrInviteNotFound))
				Expect(out).To(BeNil())
			})
		})

		It("Returns ErrCreatorNotFound when invite creator is deleted from database", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")
				invID := uuid.New()
				creatorID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:   uuid.New(),
					InviteID: invID,
				}
				m.Registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				inv, _ := invite.New(invID, creatorID, "hash", 24*time.Hour, time.Now())
				m.InviteFinderByID.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()
				m.InviteSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				// Имитируем отсутствие создателя в БД
				m.UserFinderByID.EXPECT().FindByID(ctx, creatorID).Return(nil, domain.ErrNotFound).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
					WrappedKeys: []application.WrappedKeyInput{
						{Scope: key.ScopeMain, WrappedDEK: []byte{}, WrapAlgorithm: testString},
					},
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(application.ErrCreatorNotFound))
				Expect(out).To(BeNil())
			})
		})
	})

	// --- Transaction Failures & Rollback Checks ---

	Describe("Transaction Failures & Rollback Checks", func() {
		It("Rolls back when invite redemption fails (e.g. invite already used)", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")
				invID := uuid.New()
				userID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:   userID,
					InviteID: invID,
				}
				m.Registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				// Создаем уже использованный инвайт
				inv, _ := invite.New(invID, uuid.New(), "hash", 24*time.Hour, time.Now())
				_ = inv.Redeem(uuid.New(), time.Now())

				m.InviteFinderByID.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()

				in := application.FinishInviteRegistrationInput{
					ChallengeToken:       challengeToken,
					RegistrationResponse: regResponse,
					WrappedKeys: []application.WrappedKeyInput{
						{Scope: key.ScopeMain, WrappedDEK: []byte{}, WrapAlgorithm: testString},
					},
				}

				out, err := uc.Execute(ctx, in)

				Expect(err).To(MatchError(invite.ErrAlreadyUsed))
				Expect(out).To(BeNil())
			})
		})

		It("Rolls back when WrappedKeySaver fails during transaction", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				challengeToken := testChallengeToken
				regResponse := []byte("reg-response")
				invID := uuid.New()
				creatorID := uuid.New()
				userID := uuid.New()

				regResult := application.RegistrationResult{
					UserID:     userID,
					InviteID:   invID,
					ExternalID: []byte("ext"),
					PublicKey:  []byte("pub"),
					Transports: []string{"usb"},
				}
				m.Registrator.EXPECT().Finish(ctx, challengeToken, regResponse).Return(regResult, nil).Once()

				inv, _ := invite.New(invID, creatorID, "hash", 24*time.Hour, time.Now())
				m.InviteFinderByID.EXPECT().FindByID(ctx, invID).Return(inv, nil).Once()
				m.InviteSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				creatorUser, _ := user.NewRoot(creatorID, time.Now())
				m.UserFinderByID.EXPECT().FindByID(ctx, creatorID).Return(creatorUser, nil).Once()

				m.UserSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				m.IDGenerator.EXPECT().NewID().Return(uuid.New()).Twice() // Для Credential и для Key
				m.CredentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				expectedErr := errors.New("db error saving keys")
				m.WrappedKeySaver.EXPECT().Save(ctx, mock.Anything).Return(expectedErr).Once()

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
		It("WrappedKeyInput String redaction", func() {
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
