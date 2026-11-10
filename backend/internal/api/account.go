package api

import (
	"net/http"

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
	hash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetPassword(r.Context(), u.ID, hash); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) deleteMe(w http.ResponseWriter, r *http.Request) {
	if err := a.st.DeleteUser(r.Context(), httpx.UserFrom(r).ID); err != nil {
		httpx.Fail(w, r, err)
		return
	}
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
	if in.Country == "" {
		in.Country = "GB"
	}
	if in.Line1 == "" || in.City == "" || in.Postcode == "" {
		httpx.Fail(w, r, httpx.BadRequest("Address line 1, town and postcode are required."))
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
