package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// initRouter initializes the HTTP router for the plugin.
func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/checklists/{postID}/items/{itemID}/toggle", p.handleToggleChecklistItem).Methods(http.MethodPost)

	authedRouter := apiRouter.NewRoute().Subrouter()
	authedRouter.Use(p.MattermostAuthorizationRequired)
	authedRouter.HandleFunc("/posts/{postID}/convert", p.handleConvertPostToChecklist).Methods(http.MethodPost)

	return router
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if p.router == nil {
		p.router = p.initRouter()
	}

	p.router.ServeHTTP(w, r)
}

func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.userIDFromRequest(r)
		if err != nil || userID == "" {
			writeAPIError(w, http.StatusUnauthorized, "not authorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) userIDFromRequest(r *http.Request) (string, error) {
	if userID := r.Header.Get("Mattermost-User-ID"); userID != "" {
		return userID, nil
	}

	cookie, err := r.Cookie("MMAUTHTOKEN")
	if err != nil || cookie == nil || cookie.Value == "" {
		return "", err
	}

	session, appErr := p.API.GetSession(cookie.Value)
	if appErr != nil || session == nil {
		if appErr != nil {
			return "", appErr
		}
		return "", nil
	}

	return session.UserId, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
