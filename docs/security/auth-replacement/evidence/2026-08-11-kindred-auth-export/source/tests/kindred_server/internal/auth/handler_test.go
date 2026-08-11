package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"kindred_server/internal/analytics"
	"kindred_server/internal/testutil"
)

const (
	pwd  = "fixture-secret"
	mail = "alice@example.com"
)

func TestRegisterIssuesTokenAndCapturesVerification(t *testing.T) {
	ts := testutil.NewTestServer(t)
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID            string `json:"id"`
			EmailVerified bool   `json:"emailVerified"`
		} `json:"user"`
	}
	status := ts.DoJSON(t, http.MethodPost, "/auth/register", "", map[string]any{
		"email":       mail,
		"password":    pwd,
		"displayName": "Alice",
	}, &out)
	if status != http.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	if out.Token == "" || out.User.ID == "" {
		t.Errorf("missing token or user id: %+v", out)
	}
	if out.User.EmailVerified {
		t.Errorf("new user emailVerified = true, want false")
	}
	if _, ok := ts.Mailer.VerifyTokens[mail]; !ok {
		t.Errorf("verification token not captured")
	}
}

func TestRegisterStoresInitialConsentsAndDemographics(t *testing.T) {
	ts := testutil.NewTestServer(t)
	var out struct {
		User struct {
			ID        string `json:"id"`
			City      string `json:"city"`
			BirthYear int    `json:"birthYear"`
			Gender    string `json:"gender"`
		} `json:"user"`
	}
	status := ts.DoJSON(t, http.MethodPost, "/auth/register", "", map[string]any{
		"email":       "consent@example.com",
		"password":    pwd,
		"displayName": "Consent",
		"city":        "Istanbul",
		"birthYear":   1991,
		"gender":      "non_binary",
		"consents": map[string]bool{
			"analytics":          true,
			"messaging_metadata": true,
			"precise_location":   false,
		},
	}, &out)
	if status != http.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	if out.User.City != "Istanbul" || out.User.BirthYear != 1991 || out.User.Gender != "non_binary" {
		t.Fatalf("demographics not returned: %+v", out.User)
	}
	current, err := ts.Consents.CurrentConsents(t.Context(), out.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !current["analytics"].Granted || !current["messaging_metadata"].Granted {
		t.Fatalf("initial granted consents missing: %#v", current)
	}
	if current["precise_location"].Granted || current["precise_location"].Version != 1 {
		t.Fatalf("initial precise_location consent = %#v, want denied v1", current["precise_location"])
	}
}

func TestRegisterConsentFailureCleansUserAndAllowsEmailRetry(t *testing.T) {
	ts := testutil.NewTestServer(t)
	email := "retry-consent@example.com"
	body := map[string]any{
		"email":       email,
		"password":    pwd,
		"displayName": "Retry",
		"consents": map[string]bool{
			"analytics": true,
		},
	}

	ts.Consents.PutErr = errors.New("consent ledger down")
	if status := ts.DoJSON(t, http.MethodPost, "/auth/register", "", body, nil); status != http.StatusInternalServerError {
		t.Fatalf("first register status = %d, want 500", status)
	}

	ts.Consents.PutErr = nil
	if status := ts.DoJSON(t, http.MethodPost, "/auth/register", "", body, nil); status != http.StatusCreated {
		t.Fatalf("retry register status = %d, want 201", status)
	}
}

func TestPhoneVerificationSendsSMSCode(t *testing.T) {
	ts := testutil.NewTestServer(t)
	var reg struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	phone := "+15550001234"
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "", map[string]any{
		"email":       "phone@example.com",
		"password":    pwd,
		"displayName": "Phone",
		"phone":       phone,
	}, &reg); s != http.StatusCreated {
		t.Fatalf("register status = %d", s)
	}
	var started struct {
		VerificationCode string `json:"verificationCode"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/verification/start", reg.Token, nil, &started); s != http.StatusAccepted {
		t.Fatalf("start phone status = %d", s)
	}
	if started.VerificationCode == "" {
		t.Fatalf("dev verificationCode missing")
	}
	if got := ts.SMS.PhoneCodes[phone]; got != started.VerificationCode {
		t.Fatalf("sms code = %q, response code = %q", got, started.VerificationCode)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/verification/verify", reg.Token, map[string]any{
		"code": started.VerificationCode,
	}, nil); s != http.StatusOK {
		t.Fatalf("verify phone status = %d", s)
	}
	u, _ := ts.Users.GetByID(nil, reg.User.ID)
	if !u.PhoneVerified {
		t.Fatal("phone should be verified")
	}
	if u.Phone != "+15550001234" {
		t.Fatalf("normalized phone = %q, want +15550001234", u.Phone)
	}
	bal, _ := ts.Requests.GetBalance(nil, reg.User.ID)
	if bal.Available != 40 {
		t.Fatalf("signup bonus balance = %d, want 40", bal.Available)
	}
	ledger, _, _ := ts.Requests.ListLedger(nil, reg.User.ID, 10, "")
	if len(ledger) != 1 || ledger[0].Reason != "signup_bonus" || ledger[0].Delta != 40 {
		t.Fatalf("signup bonus ledger = %+v", ledger)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/verification/start", reg.Token, nil, nil); s != http.StatusConflict {
		t.Fatalf("restart verified phone status = %d, want 409", s)
	}
	bal, _ = ts.Requests.GetBalance(nil, reg.User.ID)
	if bal.Available != 40 {
		t.Fatalf("signup bonus balance after restart = %d, want 40", bal.Available)
	}
}

func TestPhoneVerificationLocksNumberToSingleAccount(t *testing.T) {
	ts := testutil.NewTestServer(t)
	phone := "+15550005555"
	var first, second struct {
		Token string `json:"token"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "", map[string]any{
		"email":       "phone-lock-1@example.com",
		"password":    pwd,
		"displayName": "One",
		"phone":       phone,
	}, &first); s != http.StatusCreated {
		t.Fatalf("first register status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "", map[string]any{
		"email":       "phone-lock-2@example.com",
		"password":    pwd,
		"displayName": "Two",
		"phone":       phone,
	}, &second); s != http.StatusCreated {
		t.Fatalf("second register status = %d", s)
	}
	var firstCode, secondCode struct {
		VerificationCode string `json:"verificationCode"`
	}
	ts.DoJSON(t, http.MethodPost, "/me/phone/verification/start", first.Token, nil, &firstCode)
	ts.DoJSON(t, http.MethodPost, "/me/phone/verification/start", second.Token, nil, &secondCode)
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/verification/verify", first.Token, map[string]any{
		"code": firstCode.VerificationCode,
	}, nil); s != http.StatusOK {
		t.Fatalf("first verify status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/verification/verify", second.Token, map[string]any{
		"code": secondCode.VerificationCode,
	}, nil); s != http.StatusConflict {
		t.Fatalf("second verify status = %d, want 409", s)
	}
}

func TestPhoneChangeStartVerifyAndDuplicateConflict(t *testing.T) {
	ts := testutil.NewTestServer(t)
	firstID, firstToken := ts.RegisterAndLogin(t, "phone-change-1@example.com", pwd, "One")
	_, secondToken := ts.RegisterAndLogin(t, "phone-change-2@example.com", pwd, "Two")
	first, _ := ts.Users.GetByID(t.Context(), firstID)
	oldPhone := first.Phone

	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/change/start", firstToken, map[string]any{
		"newPhone": "+1 (555) 777-0000",
		"password": "wrong-password",
	}, nil); s != http.StatusUnauthorized {
		t.Fatalf("wrong password start status = %d, want 401", s)
	}

	var started struct {
		VerificationCode string `json:"verificationCode"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/change/start", firstToken, map[string]any{
		"newPhone": "+1 (555) 777-0000",
		"password": pwd,
	}, &started); s != http.StatusAccepted {
		t.Fatalf("start phone change status = %d", s)
	}
	var changed struct {
		Phone         string `json:"phone"`
		PhoneVerified bool   `json:"phoneVerified"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/change/verify", firstToken, map[string]any{
		"code": started.VerificationCode,
	}, &changed); s != http.StatusOK {
		t.Fatalf("verify phone change status = %d", s)
	}
	if changed.Phone != "+15557770000" || !changed.PhoneVerified {
		t.Fatalf("changed phone response = %+v", changed)
	}
	if ts.Users.HasPhone(oldPhone) {
		t.Fatalf("old phone lock %q still exists", oldPhone)
	}
	if !ts.Users.HasPhone("+15557770000") {
		t.Fatal("new phone lock missing")
	}

	var duplicate struct {
		VerificationCode string `json:"verificationCode"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/change/start", secondToken, map[string]any{
		"newPhone": "+1 555 777 0000",
		"password": pwd,
	}, &duplicate); s != http.StatusAccepted {
		t.Fatalf("duplicate start status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/phone/change/verify", secondToken, map[string]any{
		"code": duplicate.VerificationCode,
	}, nil); s != http.StatusConflict {
		t.Fatalf("duplicate verify status = %d, want 409", s)
	}
}

func TestDeactivationBlocksLoginAndReactivationRestoresAccess(t *testing.T) {
	ts := testutil.NewTestServer(t)
	userID, token := ts.RegisterAndLogin(t, "deactivate@example.com", pwd, "Deactivate")
	if s := ts.DoJSON(t, http.MethodPost, "/me/push-tokens", token, map[string]any{
		"token":    "push-token-1",
		"platform": "ios",
	}, nil); s != http.StatusOK {
		t.Fatalf("register push token status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/me/deactivation", token, map[string]any{
		"password": pwd,
	}, nil); s != http.StatusNoContent {
		t.Fatalf("deactivation status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodGet, "/me/points", token, nil, nil); s != http.StatusUnauthorized {
		t.Fatalf("old token status = %d, want 401", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "", map[string]any{
		"email":    "deactivate@example.com",
		"password": pwd,
	}, nil); s != http.StatusUnauthorized {
		t.Fatalf("deactivated login status = %d, want 401", s)
	}
	tokens, _ := ts.PushTokens.ListByUser(t.Context(), userID)
	if len(tokens) != 0 {
		t.Fatalf("push tokens after deactivation = %+v, want none", tokens)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/reactivate", "", map[string]any{
		"email":    "deactivate@example.com",
		"password": pwd,
	}, nil); s != http.StatusOK {
		t.Fatalf("reactivate status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "", map[string]any{
		"email":    "deactivate@example.com",
		"password": pwd,
	}, nil); s != http.StatusOK {
		t.Fatalf("reactivated login status = %d", s)
	}
}

func TestReactivateWrongPasswordUsesAccountLockout(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_, token := ts.RegisterAndLogin(t, "reactivate-lockout@example.com", pwd, "Reactivate Lockout")
	if s := ts.DoJSON(t, http.MethodPost, "/me/deactivation", token, map[string]any{
		"password": pwd,
	}, nil); s != http.StatusNoContent {
		t.Fatalf("deactivation status = %d", s)
	}

	for i := 0; i < 3; i++ {
		s := ts.DoJSON(t, http.MethodPost, "/auth/reactivate", "", map[string]any{
			"email":    "reactivate-lockout@example.com",
			"password": "wrong",
		}, nil)
		if s != http.StatusUnauthorized {
			t.Errorf("attempt %d status = %d, want 401", i, s)
		}
	}

	resp, _ := ts.Do(t, http.MethodPost, "/auth/reactivate", "", map[string]any{
		"email":    "reactivate-lockout@example.com",
		"password": pwd,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-lockout reactivate status = %d, want 401", resp.StatusCode)
	}
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter == "" {
		t.Fatal("Retry-After missing for locked reactivation")
	}
}

func TestDeleteAccountClosesFlowsRefundsAndAllowsReactivationBeforePurge(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ownerID, ownerToken := ts.RegisterAndLogin(t, "delete-owner@example.com", pwd, "Owner")
	requesterID, requesterToken := ts.RegisterAndLogin(t, "delete-requester@example.com", pwd, "Requester")
	if s := ts.DoJSON(t, http.MethodPut, "/me/consents", ownerToken, map[string]any{
		"consents": map[string]bool{
			string(analytics.PurposeAnalytics): true,
			string(analytics.PurposeMarketing): true,
		},
	}, nil); s != http.StatusOK {
		t.Fatalf("owner consent status = %d", s)
	}

	var createdItem struct {
		ID string `json:"id"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/items", ownerToken, map[string]any{
		"title":       "Chair",
		"description": "A chair",
		"category":    "furniture",
		"lat":         52.52,
		"lng":         13.405,
	}, &createdItem); s != http.StatusCreated {
		t.Fatalf("create item status = %d", s)
	}
	var createdReq struct {
		ID             string `json:"id"`
		PointsReserved int    `json:"pointsReserved"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/items/"+createdItem.ID+"/requests", requesterToken, map[string]any{}, &createdReq); s != http.StatusCreated {
		t.Fatalf("create request status = %d", s)
	}
	before, _ := ts.Requests.GetBalance(t.Context(), requesterID)

	var deleted struct {
		Status           string    `json:"status"`
		ScheduledPurgeAt time.Time `json:"scheduledPurgeAt"`
	}
	if s := ts.DoJSON(t, http.MethodDelete, "/me", ownerToken, map[string]any{
		"password": pwd,
	}, &deleted); s != http.StatusAccepted {
		t.Fatalf("delete status = %d", s)
	}
	if deleted.Status != "deletion_pending" || deleted.ScheduledPurgeAt.IsZero() {
		t.Fatalf("delete response = %+v", deleted)
	}
	current, err := ts.Consents.CurrentConsents(t.Context(), ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if current[analytics.PurposeAnalytics].Granted || current[analytics.PurposeMarketing].Granted {
		t.Fatalf("consents after delete = %#v, want revoked", current)
	}
	owner, _ := ts.Users.GetByID(t.Context(), ownerID)
	if owner.AccountStatus != "deletion_pending" {
		t.Fatalf("owner status = %q", owner.AccountStatus)
	}
	it, _ := ts.Items.GetByID(t.Context(), createdItem.ID)
	if it.Status != "cancelled" {
		t.Fatalf("item status = %q, want cancelled", it.Status)
	}
	req, _ := ts.Requests.GetByID(t.Context(), createdReq.ID)
	if req.Status != "declined" {
		t.Fatalf("request status = %q, want declined", req.Status)
	}
	after, _ := ts.Requests.GetBalance(t.Context(), requesterID)
	if after.Available != before.Available+createdReq.PointsReserved {
		t.Fatalf("requester balance after delete = %d, before=%d reserved=%d", after.Available, before.Available, createdReq.PointsReserved)
	}
	if s := ts.DoJSON(t, http.MethodGet, "/me/points", ownerToken, nil, nil); s != http.StatusUnauthorized {
		t.Fatalf("old delete token status = %d, want 401", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/reactivate", "", map[string]any{
		"email":    "delete-owner@example.com",
		"password": pwd,
	}, nil); s != http.StatusOK {
		t.Fatalf("reactivate deletion pending status = %d", s)
	}
}

func TestRegisterDuplicateEmailConflict(t *testing.T) {
	ts := testutil.NewTestServer(t)
	body := map[string]any{"email": mail, "password": pwd, "displayName": "A"}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "", body, nil); s != http.StatusCreated {
		t.Fatalf("first register status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "", body, nil); s != http.StatusConflict {
		t.Errorf("duplicate register status = %d, want 409", s)
	}
}

func TestLoginWrongPasswordAndLockout(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.RegisterAndLogin(t, mail, pwd, "Alice")

	for i := 0; i < 3; i++ {
		s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
			map[string]any{"email": mail, "password": "wrong"}, nil)
		if s != http.StatusUnauthorized {
			t.Errorf("attempt %d status = %d, want 401", i, s)
		}
	}
	// 4th attempt — even with correct password — must now be locked.
	resp, raw := ts.Do(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": pwd})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("post-lockout status = %d, want 401", resp.StatusCode)
	}
	ra := resp.Header.Get("Retry-After")
	secs, err := strconv.Atoi(ra)
	if err != nil || secs <= 0 {
		t.Errorf("Retry-After = %q, want positive integer seconds", ra)
	}
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Error.Code != "unauthorized" {
		t.Errorf("error.code = %q, want %q", env.Error.Code, "unauthorized")
	}
	if env.Error.Message != "account temporarily locked due to too many failed login attempts" {
		t.Errorf("error.message = %q (unexpected)", env.Error.Message)
	}
}

func TestVerifyEmailFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "",
		map[string]any{"email": mail, "password": pwd, "displayName": "A"}, nil); s != http.StatusCreated {
		t.Fatalf("register: %d", s)
	}
	tok := ts.Mailer.VerifyTokens[mail]
	// Bad token rejected.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/verify-email", "",
		map[string]any{"email": mail, "token": "nope"}, nil); s != http.StatusBadRequest {
		t.Errorf("bad token status = %d, want 400", s)
	}
	// Real token accepted.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/verify-email", "",
		map[string]any{"email": mail, "token": tok}, nil); s != http.StatusOK {
		t.Errorf("verify status = %d", s)
	}
}

func TestResendVerificationAlways202(t *testing.T) {
	ts := testutil.NewTestServer(t)
	// Even for unknown email, we don't leak existence.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/resend-verification", "",
		map[string]any{"email": "noone@example.com"}, nil); s != http.StatusAccepted {
		t.Errorf("status = %d, want 202", s)
	}
}

func TestPasswordForgotResetFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.RegisterAndLogin(t, mail, pwd, "A")
	if s := ts.DoJSON(t, http.MethodPost, "/auth/password/forgot", "",
		map[string]any{"email": mail}, nil); s != http.StatusAccepted {
		t.Fatalf("forgot status = %d", s)
	}
	tok := ts.Mailer.ResetTokens[mail]
	if tok == "" {
		t.Fatal("reset token not captured")
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/password/reset", "",
		map[string]any{"email": mail, "token": tok, "newPassword": "newsecret1"}, nil); s != http.StatusOK {
		t.Fatalf("reset status = %d", s)
	}
	// Old password rejected, new password works.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": pwd}, nil); s != http.StatusUnauthorized {
		t.Errorf("old password should fail, got %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": "newsecret1"}, nil); s != http.StatusOK {
		t.Errorf("new password should succeed, got %d", s)
	}
}

func TestChangePasswordRequiresOldPasswordAndRotatesLogin(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_, token := ts.RegisterAndLogin(t, mail, pwd, "A")

	if s := ts.DoJSON(t, http.MethodPost, "/auth/password/change", token,
		map[string]any{"oldPassword": "wrong", "newPassword": "newsecret1"}, nil); s != http.StatusUnauthorized {
		t.Fatalf("wrong old password status = %d, want 401", s)
	}

	if s := ts.DoJSON(t, http.MethodPost, "/auth/password/change", token,
		map[string]any{"oldPassword": pwd, "newPassword": "newsecret1"}, nil); s != http.StatusOK {
		t.Fatalf("change status = %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": pwd}, nil); s != http.StatusUnauthorized {
		t.Errorf("old password should fail, got %d", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": "newsecret1"}, nil); s != http.StatusOK {
		t.Errorf("new password should succeed, got %d", s)
	}
}

func TestRefreshDeviceIDMismatchClearsSession(t *testing.T) {
	ts := testutil.NewTestServer(t)
	// Login with a deviceID — server records the binding.
	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "",
		map[string]any{"email": mail, "password": pwd, "displayName": "A", "deviceId": "device-A"}, &login); s != http.StatusCreated {
		t.Fatalf("register: %d", s)
	}
	// Refresh from a different device must be rejected and clear the session.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/refresh", "",
		map[string]any{"refreshToken": login.RefreshToken, "deviceId": "device-B"}, nil); s != http.StatusUnauthorized {
		t.Errorf("device mismatch refresh status = %d, want 401", s)
	}
	// Session is cleared, so even the legit device can't refresh anymore.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/refresh", "",
		map[string]any{"refreshToken": login.RefreshToken, "deviceId": "device-A"}, nil); s != http.StatusUnauthorized {
		t.Errorf("post-clear refresh status = %d, want 401", s)
	}
}

func TestLogoutInvalidatesAccessToken(t *testing.T) {
	ts := testutil.NewTestServer(t)
	// Register + verify so we can issue a fresh login carrying the refresh token.
	if s := ts.DoJSON(t, http.MethodPost, "/auth/register", "",
		map[string]any{"email": mail, "password": pwd, "displayName": "A"}, nil); s != http.StatusCreated {
		t.Fatalf("register: %d", s)
	}
	verifyTok := ts.Mailer.VerifyTokens[mail]
	if s := ts.DoJSON(t, http.MethodPost, "/auth/verify-email", "",
		map[string]any{"email": mail, "token": verifyTok}, nil); s != http.StatusOK {
		t.Fatalf("verify: %d", s)
	}
	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/login", "",
		map[string]any{"email": mail, "password": pwd}, &login); s != http.StatusOK {
		t.Fatalf("login: %d", s)
	}
	if s := ts.DoJSON(t, http.MethodGet, "/me/points", login.Token, nil, nil); s != http.StatusOK {
		t.Fatalf("pre-logout status = %d, want 200", s)
	}
	if s := ts.DoJSON(t, http.MethodPost, "/auth/logout", "",
		map[string]any{"refreshToken": login.RefreshToken}, nil); s != http.StatusOK {
		t.Fatalf("logout status = %d", s)
	}
	// Same bearer must now be rejected — TokenVersion was bumped.
	if s := ts.DoJSON(t, http.MethodGet, "/me/points", login.Token, nil, nil); s != http.StatusUnauthorized {
		t.Errorf("post-logout status = %d, want 401", s)
	}
}

func TestRateLimit(t *testing.T) {
	ts := testutil.NewTestServerWithLimiter(t, 2, time.Minute)
	body := map[string]any{"email": "rate@example.com", "password": pwd, "displayName": "R"}
	// Two attempts allowed.
	for i := 0; i < 2; i++ {
		ts.DoJSON(t, http.MethodPost, "/auth/login", "", body, nil)
	}
	// Third attempt must hit the limiter.
	s := ts.DoJSON(t, http.MethodPost, "/auth/login", "", body, nil)
	if s != http.StatusTooManyRequests {
		t.Errorf("rate-limit status = %d, want 429", s)
	}
}

func TestOpenAPIUserSchemaDocumentsEmailVerified(t *testing.T) {
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "emailVerified: { type: boolean }") {
		t.Fatal("OpenAPI User schema missing emailVerified boolean field")
	}
}
