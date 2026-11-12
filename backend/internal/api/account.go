package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.st.UserByID(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrUnauthorized)
		return
	}
	httpx.JSON(w, 200, u)
}

func (a *API) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FirstName         *string `json:"firstName"`
		LastName          *string `json:"lastName"`
		Phone             *string `json:"phone"`
		MarketingEmail    *bool   `json:"marketingEmail"`
		MarketingSMS      *bool   `json:"marketingSms"`
		MarketingWhatsApp *bool   `json:"marketingWhatsapp"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := httpx.CheckLengths(httpx.Limit{Name: "First name", Value: deref(in.FirstName), Max: 60},
		httpx.Limit{Name: "Last name", Value: deref(in.LastName), Max: 60}, httpx.Limit{Name: "Phone", Value: deref(in.Phone), Max: 30}); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if in.FirstName != nil && strings.TrimSpace(*in.FirstName) == "" {
		httpx.Fail(w, r, httpx.Unprocessable("missing_name", "Enter your first name."))
		return
	}
	cols := map[string]any{}
	if in.FirstName != nil {
		cols["first_name"] = *in.FirstName
	}
	if in.LastName != nil {
		cols["last_name"] = *in.LastName
	}
	if in.Phone != nil {
		cols["phone"] = *in.Phone
	}
	if in.MarketingEmail != nil {
		cols["marketing_email"] = *in.MarketingEmail
	}
	if in.MarketingSMS != nil {
		cols["marketing_sms"] = *in.MarketingSMS
	}
	if in.MarketingWhatsApp != nil {
		cols["marketing_whatsapp"] = *in.MarketingWhatsApp
	}
	u, err := a.st.UpdateUser(r.Context(), httpx.UserFrom(r).ID, cols)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, u)
}

// changePassword signs out every other device (all refresh tokens revoked)
// and starts a fresh session for this one.
func (a *API) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct{ CurrentPassword, NewPassword string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u, err := a.st.UserByID(r.Context(), httpx.UserFrom(r).ID)
	if err != nil || !auth.CheckPassword(u.PasswordHash, in.CurrentPassword) {
		httpx.Fail(w, r, httpx.E(401, "wrong_password", "Your current password is incorrect."))
		return
	}
	if err := auth.ValidatePassword(in.NewPassword, u.Email); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetPassword(r.Context(), u.ID, hash); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	_ = a.st.RevokeAllRefresh(r.Context(), u.ID)
	res, err := a.issue(w, r, u, "")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, res)
}

// deleteMe erases the account (UK GDPR). The password is required so a
// stolen access token alone cannot delete someone's account.
func (a *API) deleteMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	_ = httpx.DecodeLoose(r, &in)
	u, err := a.st.UserByID(r.Context(), httpx.UserFrom(r).ID)
	if err != nil || !auth.CheckPassword(u.PasswordHash, in.Password) {
		httpx.Fail(w, r, httpx.E(401, "wrong_password", "Enter your current password to delete your account."))
		return
	}
	if err := a.st.DeleteUser(r.Context(), u.ID); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.setRefreshCookie(w, "", -1)
	httpx.NoContent(w)
}

func (a *API) exportMe(w http.ResponseWriter, r *http.Request) {
	data, err := a.st.ExportUserData(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="celestial-oud-my-data.json"`)
	httpx.JSON(w, 200, data)
}

func (a *API) myOrders(w http.ResponseWriter, r *http.Request) {
	os, err := a.st.ListOrdersForUser(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, os)
}

func (a *API) myAddresses(w http.ResponseWriter, r *http.Request) {
	as, err := a.st.ListAddresses(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, as)
}

func (a *API) saveAddress(w http.ResponseWriter, r *http.Request) {
	var in store.Address
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		n, err := pathInt(r, "id")
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		in.ID = n
	}
	if err := validateAddress(&in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	out, err := a.st.UpsertAddress(r.Context(), httpx.UserFrom(r).ID, in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, out)
}

func (a *API) deleteAddress(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeleteAddress(r.Context(), httpx.UserFrom(r).ID, id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) wishlist(w http.ResponseWriter, r *http.Request) {
	slugs, err := a.st.Wishlist(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, slugs)
}

func (a *API) toggleWishlist(w http.ResponseWriter, r *http.Request) {
	added, err := a.st.ToggleWishlist(r.Context(), httpx.UserFrom(r).ID, chi.URLParam(r, "slug"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, map[string]bool{"saved": added})
}

func (a *API) scentProfile(w http.ResponseWriter, r *http.Request) {
	p, err := a.st.ScentProfile(r.Context(), httpx.UserFrom(r).ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, p)
}

func (a *API) saveScentProfile(w http.ResponseWriter, r *http.Request) {
	var in store.ScentProfile
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SaveScentProfile(r.Context(), httpx.UserFrom(r).ID, in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}
