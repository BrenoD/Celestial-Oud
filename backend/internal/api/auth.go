package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

// Sessions: a 15-minute access JWT returned in the body (kept in memory by
// the SPA) and a rotating 30-day refresh token in an HttpOnly, Secure,
// SameSite cookie scoped to /v1/auth — script on the page can never read it.
type authResponse struct {
	Token     string      `json:"token"`
	ExpiresAt string      `json:"expiresAt"`
	User      *store.User `json:"user"`
}

func (a *API) cookieName() string {
	if a.cfg.CookieSecure {
		return "__Secure-co_rt"
	}
	return "co_rt"
}

func (a *API) setRefreshCookie(w http.ResponseWriter, value string, maxAge int) {
	mode := http.SameSiteStrictMode
	switch strings.ToLower(a.cfg.CookieSameSite) {
	case "lax":
		mode = http.SameSiteLaxMode
	case "none":
		mode = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name: a.cookieName(), Value: value, Path: "/v1/auth", MaxAge: maxAge,
		HttpOnly: true, Secure: a.cfg.CookieSecure || mode == http.SameSiteNoneMode, SameSite: mode,
	})
}

// trustedOrigin guards the cookie-authenticated endpoints against CSRF: the
// browser's Origin must be one of the storefront origins.
func (a *API) trustedOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	return o != "" && slices.Contains(a.cfg.CORSOrigins, o)
}

func (a *API) issue(w http.ResponseWriter, r *http.Request, u *store.User, family string) (*authResponse, error) {
	tok, exp, err := a.auth.IssueAccess(u.ID, u.Email, u.Role)
	if err != nil {
		return nil, err
	}
	plain, hash := auth.NewOpaqueToken()
	if err := a.st.SaveRefresh(r.Context(), u.ID, hash, r.UserAgent(), family, a.auth.RefreshTTL()); err != nil {
		return nil, err
	}
	a.setRefreshCookie(w, plain, int(a.auth.RefreshTTL().Seconds()))
	a.st.TouchLogin(r.Context(), u.ID)
	return &authResponse{Token: tok, ExpiresAt: exp.UTC().Format("2006-01-02T15:04:05Z"), User: u}, nil
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email, Password, FirstName, LastName string
		MarketingOptIn                       bool `json:"marketingOptIn"`
	}
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	email, ok := httpx.NormEmail(in.Email)
	if !ok {
		httpx.Fail(w, r, httpx.InvalidEmail)
		return
	}
	in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
	if in.FirstName == "" {
		httpx.Fail(w, r, httpx.Unprocessable("missing_name", "Enter your first name."))
		return
	}
	if err := httpx.CheckLengths(httpx.Limit{Name: "First name", Value: in.FirstName, Max: 60}, httpx.Limit{Name: "Last name", Value: in.LastName, Max: 60}); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := auth.ValidatePassword(in.Password, email); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.CreateUser(r.Context(), email, hash, in.FirstName, in.LastName, in.MarketingOptIn)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	res, err := a.issue(w, r, u, "")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Send(r.Context(), "welcome", u.Email, map[string]any{"name": u.FirstName})
	httpx.JSON(w, 201, res)
}

var errCredentials = httpx.E(401, "invalid_credentials", "Incorrect email or password.")

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	email, _ := httpx.NormEmail(in.Email)
	failKey := "login-fail:" + subjectKey(email)
	l := limits["login-fail"]
	if a.st.Hits(r.Context(), failKey, l.window) >= l.n {
		// Account locked for the rest of the window; same wording whether or
		// not the account exists, so this cannot be used to enumerate emails.
		auth.CheckPassword("", in.Password)
		httpx.Fail(w, r, httpx.E(429, "rate_limited", "Too many attempts. Please wait 15 minutes or reset your password."))
		return
	}
	u, err := a.st.UserByEmail(r.Context(), email)
	hash := ""
	if err == nil {
		hash = u.PasswordHash
	}
	// CheckPassword spends a full bcrypt even for unknown emails (no timing oracle).
	if !auth.CheckPassword(hash, in.Password) || err != nil {
		_, _, _ = a.st.Hit(r.Context(), failKey, l.n, l.window)
		httpx.Fail(w, r, errCredentials)
		return
	}
	a.st.ClearHits(r.Context(), failKey)
	res, err := a.issue(w, r, u, "")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, res)
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	if !a.trustedOrigin(r) {
		httpx.Fail(w, r, httpx.ErrForbidden)
		return
	}
	c, err := r.Cookie(a.cookieName())
	if err != nil || c.Value == "" {
		httpx.Fail(w, r, errSessionExpired)
		return
	}
	uid, family, err := a.st.ConsumeRefresh(r.Context(), auth.HashToken(c.Value))
	if err != nil {
		a.setRefreshCookie(w, "", -1)
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.UserByID(r.Context(), uid)
	if err != nil {
		a.setRefreshCookie(w, "", -1)
		httpx.Fail(w, r, errSessionExpired)
		return
	}
	res, err := a.issue(w, r, u, family)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, res)
}

var errSessionExpired = httpx.E(401, "invalid_refresh", "Your session has expired. Please sign in again.")

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if !a.trustedOrigin(r) {
		httpx.Fail(w, r, httpx.ErrForbidden)
		return
	}
	if c, err := r.Cookie(a.cookieName()); err == nil && c.Value != "" {
		a.st.RevokeRefresh(r.Context(), auth.HashToken(c.Value))
	}
	a.setRefreshCookie(w, "", -1)
	httpx.NoContent(w)
}

func (a *API) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	email, ok := httpx.NormEmail(in.Email)
	l := limits["forgot-email"]
	// Per-address cap stops the endpoint being used to flood someone's inbox;
	// over the cap we still answer 204 so nothing is revealed.
	if within, _, _ := a.st.Hit(r.Context(), "forgot-email:"+subjectKey(email), l.n, l.window); ok && within {
		if u, err := a.st.UserByEmail(r.Context(), email); err == nil {
			plain, hash := auth.NewOpaqueToken()
			if err := a.st.SaveReset(r.Context(), u.ID, hash); err == nil {
				a.mail.Send(r.Context(), "password_reset", u.Email, map[string]any{"url": a.cfg.PublicURL + "/account/reset?token=" + plain})
			}
		}
	}
	httpx.NoContent(w)
}

func (a *API) reset(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := auth.ValidatePassword(in.Password, ""); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	uid, err := a.st.ConsumeReset(r.Context(), auth.HashToken(in.Token))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.UserByID(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := auth.ValidatePassword(in.Password, u.Email); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetPassword(r.Context(), uid, hash); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	_ = a.st.RevokeAllRefresh(r.Context(), uid)
	a.st.ClearHits(r.Context(), "login-fail:"+subjectKey(u.Email))
	httpx.NoContent(w)
}
