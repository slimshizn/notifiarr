package plex

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"golift.io/starr"
)

// HandleSessions provides a web handler to the notifiarr client that returns
// the current Plex sessions. The handler satisfies apps.APIHandler, sorry.
// @Description  Returns Plex sessions.
// @Summary      Retrieve Plex sessions.
// @Tags         Plex
// @Produce      json
// @Success      200  {object} apps.Respond.apiResponse{message=Sessions} "contains app info included appStatus"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "Plex error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/plex/1/sessions [get]
func (s *Server) HandleSessions(r *http.Request) (int, interface{}) {
	plexID, _ := r.Context().Value(starr.Plex).(int)

	sessions, err := s.GetSessionsWithContext(r.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to get sessions (%d): %w", plexID, err)
	}

	return http.StatusOK, sessions
}

// HandleKillSession provides a web handler to the notifiarr client allows
// notifiarr.com (via Discord request) to end a Plex session.
// @Description  Kills a Plex session by ID and sends a message to the user.
// @Summary      Kill a Plex session.
// @Tags         Plex
// @Produce      json
// @Param        sessionId  query   string  true  "Plex session ID"
// @Param        reason     query   string  true  "Reason the session is being terminated. Sent to the user."
// @Success      200  {object} apps.Respond.apiResponse{message=string} "success"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "Plex error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/plex/1/kill [get]
func (s *Server) HandleKillSession(r *http.Request) (int, interface{}) {
	var (
		ctx       = r.Context()
		plexID, _ = ctx.Value(starr.Plex).(int)
		sessionID = mux.Vars(r)["sessionId"]
		reason    = mux.Vars(r)["reason"]
	)

	_, err := s.KillSessionWithContext(ctx, sessionID, reason)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to kill session (%s@%d): %w", sessionID, plexID, err)
	}

	return http.StatusOK, fmt.Sprintf("kilt session '%s' with reason: %s", sessionID, reason)
}

// @Description  Returns the Plex Library Directory.
// @Summary      Retrieve the Plex Library Directory.
// @Tags         Plex
// @Produce      json
// @Success      200  {object} apps.Respond.apiResponse{message=SectionDirectory} "Plex Library Directory"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "Plex error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/plex/1/directory [get]
func (s *Server) HandleDirectory(r *http.Request) (int, interface{}) {
	plexID, _ := r.Context().Value(starr.Plex).(int)

	directory, err := s.GetDirectoryWithContext(r.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("directory request failed (%d): %w", plexID, err)
	}

	return http.StatusOK, directory
}

// @Description  Empties the Plex library trash for the provided library key. Get the library key from the Directory.
// @Summary      Empty Plex Trash
// @Tags         Plex
// @Produce      json
// @Param        libraryKey   path    string true  "Plex Library Section Key"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "Plex error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/plex/1/emptytrash/{libraryKey} [get]
func (s *Server) HandleEmptyTrash(r *http.Request) (int, interface{}) {
	plexID, _ := r.Context().Value(starr.Plex).(int)

	body, err := s.EmptyTrashWithContext(r.Context(), mux.Vars(r)["key"])
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("trash empty failed (%d): %w", plexID, err)
	}

	return http.StatusOK, "ok: " + string(body)
}

// @Description  Marks a movie or show or audio track as watched.
// @Summary      Mark a Plex item as watched.
// @Tags         Plex
// @Produce      json
// @Param        itemKey  path    string true  "Plex Item Key"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "Plex error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/plex/1/markwatched/{itemKey} [get]
func (s *Server) HandleMarkWatched(r *http.Request) (int, interface{}) {
	plexID, _ := r.Context().Value(starr.Plex).(int)

	body, err := s.MarkPlayedWithContext(r.Context(), mux.Vars(r)["key"])
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("mark watch failed (%d): %w", plexID, err)
	}

	return http.StatusOK, "ok: " + string(body)
}
