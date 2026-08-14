package webauthnadapter

import "fmt"

// SECURITY: never log this field
func (p loginChallengePayload) String() string {
	return "loginChallengePayload{SessionData: ***REDACTED***}"
}

// SECURITY: never log this field
func (p challengePayload) String() string {
	return fmt.Sprintf("challengePayload{SessionData: ***REDACTED***, InviteID: %v, UserID: %v}", p.InviteID, p.UserID)
}
