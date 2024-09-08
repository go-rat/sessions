package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-rat/sessions"
)

// StartSession is an example middleware that starts a session for each request.
// If this middleware not suitable for your application, you can create your own.
func StartSession(manager *sessions.Manager, driver ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if session exists
			_, ok := r.Context().Value(sessions.CtxKey).(*sessions.Session)
			if ok {
				next.ServeHTTP(w, r)
				return
			}

			// Build session
			s, err := manager.BuildSession(sessions.CookieName, driver...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Try to get and decode session ID from cookie
			sessionID := s.GetID()
			if cookie, err := r.Cookie(s.GetName()); err == nil {
				if err = manager.Codec.Decode(s.GetName(), cookie.Value, &sessionID); err == nil {
					s.SetID(sessionID)
				}
			}

			// Start session
			s.Start()
			r = r.WithContext(context.WithValue(r.Context(), sessions.CtxKey, s)) //nolint:staticcheck

			// Encode session ID
			if encoded, err := manager.Codec.Encode(s.GetName(), s.GetID()); err == nil {
				sessionID = encoded
			}

			// Set session cookie in response
			http.SetCookie(w, &http.Cookie{
				Name:        s.GetName(),
				Value:       sessionID,
				Expires:     time.Now().Add(time.Duration(manager.Lifetime) * time.Minute),
				Secure:      true,
				HttpOnly:    true,
				SameSite:    http.SameSiteLaxMode,
				Partitioned: true,
			})

			// Continue processing request
			next.ServeHTTP(w, r)

			// Save session
			if err = s.Save(); err != nil {
				log.Printf("session save error: %v", err)
			}

			// Release session
			manager.ReleaseSession(s)
		})
	}
}
