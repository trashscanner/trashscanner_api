package api

import (
	"net/http"

	"github.com/trashscanner/trashscanner_api/internal/auth"
)

const (
	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
)

func setAuthCookies(w http.ResponseWriter, tokens *auth.TokenPair) {
	accessCookie := &http.Cookie{
		Name:     accessCookieName,
		Value:    tokens.Access,
		HttpOnly: true,
		Secure:   true,
	}
	refreshCookie := &http.Cookie{
		Name:     refreshCookieName,
		Value:    tokens.Refresh,
		HttpOnly: true,
		Secure:   true,
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

func getAccessCookie(r *http.Request) (string, error) {
	accessCookie, err := r.Cookie(accessCookieName)
	if err != nil {
		return "", err
	}
	return accessCookie.Value, nil
}

func getRefreshFromCookie(r *http.Request) (string, error) {
	refreshCookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		return "", err
	}
	return refreshCookie.Value, nil
}

func clearAuthCookies(w http.ResponseWriter) {
	accessCookie := &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}
	refreshCookie := &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}
