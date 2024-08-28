package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-rat/session/contract"
)

var (
	CtxKey     = "session" // session context key
	CookieName = "session" // session cookie name
)

// StartSession is an example middleware that starts a session for each request.
// If this middleware not suitable for your application, you can create your own.
func StartSession(manager contract.Manager, driver string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if session exists
			_, ok := r.Context().Value(CtxKey).(contract.Session)
			if ok {
				next.ServeHTTP(w, r)
				return
			}

			// Build session
			s, err := manager.BuildSession(CookieName, driver)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			cookie, _ := r.Cookie(s.GetName())
			s.SetID(cookie.Value)

			// Start session
			s.Start()
			r = r.WithContext(context.WithValue(r.Context(), CtxKey, s))

			// Set session cookie in response
			http.SetCookie(w, &http.Cookie{
				Name:     s.GetName(),
				Value:    s.GetID(),
				Expires:  time.Now().Add(time.Duration(120) * time.Minute),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			// Continue processing request
			next.ServeHTTP(w, r)

			// Save session
			if err = s.Save(); err != nil {
				log.Printf("session save error: %v", err)
			}
		})
	}
}
