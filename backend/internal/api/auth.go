package api

import (
	"net/http"
	"strings"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

type authResponse struct {
	Token        string      `json:"token"`
	ExpiresAt    string      `json:"expiresAt"`
	RefreshToken string      `json:"refreshToken"`
	User         *store.User `json:"user"`
}

func (a *API) issue(r *http.Request, u *store.User) (*authResponse, error) {
	tok, exp, err := a.auth.IssueAccess(u.ID, u.Email, u.Role)
	if err != nil {
		return nil, err
	}
	plain, hash := auth.NewOpaqueToken()
	if err := a.st.SaveRefresh(r.Context(), u.ID, hash, r.UserAgent(), a.auth.RefreshTTL()); err != nil {
		return nil, err
	}
	a.st.TouchLogin(r.Context(), u.ID)
	return &authResponse{Token: tok, ExpiresAt: exp.UTC().Format("2006-01-02T15:04:05Z"), RefreshToken: plain, User: u}, nil
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
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") || strings.TrimSpace(in.FirstName) == "" {
		httpx.Fail(w, r, httpx.BadRequest("Email and first name are required."))
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.CreateUser(r.Context(), in.Email, hash, strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName), in.MarketingOptIn)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	res, err := a.issue(r, u)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Send(r.Context(), "welcome", u.Email, map[string]any{"name": u.FirstName})
	httpx.JSON(w, 201, res)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.UserByEmail(r.Context(), in.Email)
	// Same message for unknown email and wrong password: no enumeration.
	if err != nil || u.PasswordHash == "" || !auth.CheckPassword(u.PasswordHash, in.Password) {
		httpx.Fail(w, r, httpx.E(401, "invalid_credentials", "Incorrect email or password."))
		return
	}
	res, err := a.issue(r, u)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, res)
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	uid, err := a.st.ConsumeRefresh(r.Context(), auth.HashToken(in.RefreshToken))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.UserByID(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrUnauthorized)
		return
	}
	res, err := a.issue(r, u)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, res)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = httpx.DecodeLoose(r, &in)
	if in.RefreshToken != "" {
		_, _ = a.st.ConsumeRefresh(r.Context(), auth.HashToken(in.RefreshToken))
	}
	httpx.NoContent(w)
}

func (a *API) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if u, err := a.st.UserByEmail(r.Context(), in.Email); err == nil {
		plain, hash := auth.NewOpaqueToken()
		if err := a.st.SaveReset(r.Context(), u.ID, hash); err == nil {
			a.mail.Send(r.Context(), "password_reset", u.Email, map[string]any{"url": a.cfg.PublicURL + "/account/reset?token=" + plain})
		}
	}
	// Always 204: do not reveal whether the email exists.
	httpx.NoContent(w)
}

func (a *API) reset(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	uid, err := a.st.ConsumeReset(r.Context(), auth.HashToken(in.Token))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetPassword(r.Context(), uid, hash); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	_ = a.st.RevokeAllRefresh(r.Context(), uid)
	httpx.NoContent(w)
}
