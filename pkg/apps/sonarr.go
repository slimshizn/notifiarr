//nolint:dupl
package apps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"github.com/gorilla/mux"
	"golift.io/starr"
	"golift.io/starr/debuglog"
	"golift.io/starr/sonarr"
)

// sonarrHandlers is called once on startup to register the web API paths.
func (a *Apps) sonarrHandlers() {
	a.HandleAPIpath(starr.Sonarr, "/add", sonarrAddSeries, "POST")
	a.HandleAPIpath(starr.Sonarr, "/check/{tvdbid:[0-9]+}", sonarrCheckSeries, "GET")
	a.HandleAPIpath(starr.Sonarr, "/get/{seriesid:[0-9]+}", sonarrGetSeries, "GET")
	a.HandleAPIpath(starr.Sonarr, "/getEpisodes/{seriesid:[0-9]+}", sonarrGetEpisodes, "GET")
	a.HandleAPIpath(starr.Sonarr, "/unmonitor/{episodeid:[0-9]+}", sonarrUnmonitorEpisode, "GET")
	a.HandleAPIpath(starr.Sonarr, "/languageProfiles", sonarrLangProfiles, "GET")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfiles", sonarrGetQualityProfiles, "GET")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfile", sonarrGetQualityProfile, "GET")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfile", sonarrAddQualityProfile, "POST")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfile/{profileID:[0-9]+}", sonarrUpdateQualityProfile, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfile/{profileID:[0-9]+}", sonarrDeleteQualityProfile, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/qualityProfiles/all", sonarrDeleteAllQualityProfiles, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/releaseProfiles", sonarrGetReleaseProfiles, "GET")
	a.HandleAPIpath(starr.Sonarr, "/releaseProfile", sonarrAddReleaseProfile, "POST")
	a.HandleAPIpath(starr.Sonarr, "/releaseProfile/{profileID:[0-9]+}", sonarrUpdateReleaseProfile, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/releaseProfile/{profileID:[0-9]+}", sonarrDeleteReleaseProfile, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/releaseProfiles/all", sonarrDeleteAllReleaseProfiles, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/customformats", sonarrGetCustomFormats, "GET")
	a.HandleAPIpath(starr.Sonarr, "/customformats", sonarrAddCustomFormat, "POST")
	a.HandleAPIpath(starr.Sonarr, "/customformats/{cfid:[0-9]+}", sonarrUpdateCustomFormat, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/customformats/{cfid:[0-9]+}", sonarrDeleteCustomFormat, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/customformats/all", sonarrDeleteAllCustomFormats, "DELETE")
	a.HandleAPIpath(starr.Sonarr, "/qualitydefinitions", sonarrGetQualityDefinitions, "GET")
	a.HandleAPIpath(starr.Sonarr, "/qualitydefinition", sonarrUpdateQualityDefinition, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/rootFolder", sonarrRootFolders, "GET")
	a.HandleAPIpath(starr.Sonarr, "/naming", sonarrGetNaming, "GET")
	a.HandleAPIpath(starr.Sonarr, "/naming", sonarrUpdateNaming, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/search/{query}", sonarrSearchSeries, "GET")
	a.HandleAPIpath(starr.Sonarr, "/tag", sonarrGetTags, "GET")
	a.HandleAPIpath(starr.Sonarr, "/tag/{tid:[0-9]+}/{label}", sonarrUpdateTag, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/tag/{label}", sonarrSetTag, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/update", sonarrUpdateSeries, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/seasonPass", sonarrSeasonPass, "POST")
	a.HandleAPIpath(starr.Sonarr, "/command/{commandid:[0-9]+}", sonarrStatusCommand, "GET")
	a.HandleAPIpath(starr.Sonarr, "/command", sonarrTriggerCommand, "POST")
	a.HandleAPIpath(starr.Sonarr, "/command/search/{seriesid:[0-9]+}", sonarrTriggerSearchSeries, "GET")
	a.HandleAPIpath(starr.Sonarr, "/notification", sonarrGetNotifications, "GET")
	a.HandleAPIpath(starr.Sonarr, "/notification", sonarrUpdateNotification, "PUT")
	a.HandleAPIpath(starr.Sonarr, "/notification", sonarrAddNotification, "POST")
}

// SonarrConfig represents the input data for a Sonarr server.
type SonarrConfig struct {
	*sonarr.Sonarr `toml:"-" xml:"-" json:"-"`
	extraConfig
	*starr.Config
	errorf func(string, ...interface{}) `toml:"-" xml:"-" json:"-"`
}

func getSonarr(r *http.Request) *sonarr.Sonarr {
	app, _ := r.Context().Value(starr.Sonarr).(*SonarrConfig)
	return app.Sonarr
}

// Enabled returns true if the Sonarr instance is enabled and usable.
func (s *SonarrConfig) Enabled() bool {
	return s != nil && s.Config != nil && s.URL != "" && s.APIKey != "" && s.Timeout.Duration >= 0
}

func (a *Apps) setupSonarr() error {
	for idx, app := range a.Sonarr {
		if app.Config == nil || app.Config.URL == "" {
			return fmt.Errorf("%w: missing url: Sonarr config %d", ErrInvalidApp, idx+1)
		} else if !strings.HasPrefix(app.Config.URL, "http://") && !strings.HasPrefix(app.Config.URL, "https://") {
			return fmt.Errorf("%w: URL must begin with http:// or https://: Sonarr config %d", ErrInvalidApp, idx+1)
		}

		if a.Logger.DebugEnabled() {
			app.Config.Client = starr.ClientWithDebug(app.Timeout.Duration, app.ValidSSL, debuglog.Config{
				MaxBody: a.MaxBody,
				Debugf:  a.Debugf,
				Caller:  metricMakerCallback(string(starr.Sonarr)),
			})
		} else {
			app.Config.Client = starr.Client(app.Timeout.Duration, app.ValidSSL)
			app.Config.Client.Transport = NewMetricsRoundTripper(starr.Sonarr.String(), nil)
		}

		app.errorf = a.Errorf
		app.URL = strings.TrimRight(app.URL, "/")
		app.Sonarr = sonarr.New(app.Config)
	}

	return nil
}

// @Description  Adds a new Series to Sonarr.
// @Summary      Add Sonarr Series
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.AddSeriesInput true "new item content"
// @Success      201  {object} apps.Respond.apiResponse{message=sonarr.Series} "series content"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "bad json payload"
// @Failure      409  {object} apps.Respond.apiResponse{message=string} "item already exists"
// @Failure      422  {object} apps.Respond.apiResponse{message=string} "no item ID provided"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error during check"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error during add"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/add [post]
// @Security     ApiKeyAuth
func sonarrAddSeries(req *http.Request) (int, interface{}) {
	var payload sonarr.AddSeriesInput
	// Extract payload and check for TVDB ID.
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	} else if payload.TvdbID == 0 {
		return http.StatusUnprocessableEntity, fmt.Errorf("0: %w", ErrNoTVDB)
	}

	// Check for existing series.
	m, err := getSonarr(req).GetSeriesContext(req.Context(), payload.TvdbID)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("checking series: %w", err)
	} else if len(m) > 0 {
		return http.StatusConflict, sonarrData(m[0])
	}

	series, err := getSonarr(req).AddSeriesContext(req.Context(), &payload)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("adding series: %w", err)
	}

	return http.StatusCreated, series
}

func sonarrData(series *sonarr.Series) map[string]interface{} {
	hasFile := false
	if series.Statistics != nil {
		hasFile = series.Statistics.SizeOnDisk > 0
	}

	return map[string]interface{}{
		"id":        series.ID,
		"hasFile":   hasFile,
		"monitored": series.Monitored,
		"tags":      series.Tags,
	}
}

// @Description  Checks if a Sonarr Series already exists.
// @Summary      Check Sonarr Series Existence
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        tvdbid    path   int64  true  "TVDB ID"
// @Success      201  {object} apps.Respond.apiResponse{message=string} "series does not exist"
// @Failure      409  {object} apps.Respond.apiResponse{message=string} "item already exists"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/check/{tvdbid} [get]
// @Security     ApiKeyAuth
func sonarrCheckSeries(req *http.Request) (int, interface{}) {
	tvdbid, _ := strconv.ParseInt(mux.Vars(req)["tvdbid"], mnd.Base10, mnd.Bits64)
	// Check for existing series.
	m, err := getSonarr(req).GetSeriesContext(req.Context(), tvdbid)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("checking series: %w", err)
	} else if len(m) > 0 {
		return http.StatusConflict, sonarrData(m[0])
	}

	return http.StatusOK, http.StatusText(http.StatusNotFound)
}

// @Description  Returns a Sonarr Series by ID.
// @Summary      Get Sonarr Series
// @Tags         Sonarr
// @Produce      json
// @Param        instance   path   int64  true  "instance ID"
// @Param        seriesID   path   int64  true  "Series ID"
// @Success      201  {object} apps.Respond.apiResponse{message=sonarr.Series} "series content"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/get/{seriesID} [get]
// @Security     ApiKeyAuth
func sonarrGetSeries(req *http.Request) (int, interface{}) {
	seriesID, _ := strconv.ParseInt(mux.Vars(req)["seriesid"], mnd.Base10, mnd.Bits64)

	series, err := getSonarr(req).GetSeriesByIDContext(req.Context(), seriesID)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("checking series: %w", err)
	}

	return http.StatusOK, series
}

// @Description  Returns a Sonarr Series Episodes by Series ID.
// @Summary      Get Sonarr Series Episodes
// @Tags         Sonarr
// @Produce      json
// @Param        instance   path   int64  true  "instance ID"
// @Param        seriesID   path   int64  true  "Series ID"
// @Success      201  {object} apps.Respond.apiResponse{message=[]sonarr.Episode} "episodes content"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/getEpisodes/{seriesID} [get]
// @Security     ApiKeyAuth
func sonarrGetEpisodes(req *http.Request) (int, interface{}) {
	seriesID, _ := strconv.ParseInt(mux.Vars(req)["seriesid"], mnd.Base10, mnd.Bits64)

	episodes, err := getSonarr(req).GetSeriesEpisodesContext(req.Context(), seriesID)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("checking series: %w", err)
	}

	return http.StatusOK, episodes
}

// @Description  Unmonnitors and returns a Sonarr Series Episode.
// @Summary      Unmonnitors Sonarr Series Episode
// @Tags         Sonarr
// @Produce      json
// @Param        instance    path   int64  true  "instance ID"
// @Param        episodeID   path   int64  true  "Episode ID"
// @Success      201  {object} apps.Respond.apiResponse{message=sonarr.Episode} "episode content"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/unmonitor/{episodeID} [get]
// @Security     ApiKeyAuth
func sonarrUnmonitorEpisode(req *http.Request) (int, interface{}) {
	episodeID, _ := strconv.ParseInt(mux.Vars(req)["episodeid"], mnd.Base10, mnd.Bits64)

	episodes, err := getSonarr(req).MonitorEpisodeContext(req.Context(), []int64{episodeID}, false)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("checking series: %w", err)
	} else if len(episodes) != 1 {
		return http.StatusServiceUnavailable, fmt.Errorf("%w (%d): %v", ErrWrongCount, len(episodes), episodes)
	}

	return http.StatusOK, episodes[0]
}

// @Description  Trigger an Internet search for a Sonarr Series.
// @Summary      Search for Sonarr Series
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        seriesID   path   int64  true  "Series ID"
// @Success      201  {object} apps.Respond.apiResponse{message=string} "search status"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/command/search/{seriesID} [get]
// @Security     ApiKeyAuth
func sonarrTriggerSearchSeries(req *http.Request) (int, interface{}) {
	seriesID, _ := strconv.ParseInt(mux.Vars(req)["seriesid"], mnd.Base10, mnd.Bits64)

	output, err := getSonarr(req).SendCommandContext(req.Context(), &sonarr.CommandRequest{
		Name:     "SeriesSearch",
		SeriesID: seriesID,
	})
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("triggering series search: %w", err)
	}

	return http.StatusOK, output.Status
}

// @Description  Execute any command in Sonarr.
// @Summary      Execute Sonarr Command
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.CommandRequest true "command content, must include series ID"
// @Success      201  {object} apps.Respond.apiResponse{message=sonarr.CommandResponse} "command response"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "invalid json input"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/command [post]
// @Security     ApiKeyAuth
func sonarrTriggerCommand(req *http.Request) (int, interface{}) {
	var command sonarr.CommandRequest

	err := json.NewDecoder(req.Body).Decode(&command)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding command payload: %w", err)
	}

	output, err := getSonarr(req).SendCommandContext(req.Context(), &command)
	if err != nil {
		return http.StatusServiceUnavailable,
			fmt.Errorf("triggering command '%s' on series %d: %w", command.Name, command.SeriesID, err)
	}

	return http.StatusOK, output
}

// @Description  Check the status of an executed Sonarr Command.
// @Summary      Sonar Command Status
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        commandID   path   int64  true  "Command ID returned by executing a command"
// @Success      201  {object} apps.Respond.apiResponse{message=sonarr.CommandResponse} "command status"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/command/{commandID} [get]
// @Security     ApiKeyAuth
func sonarrStatusCommand(req *http.Request) (int, interface{}) {
	commandID, _ := strconv.ParseInt(mux.Vars(req)["commandid"], mnd.Base10, mnd.Bits64)

	output, err := getSonarr(req).GetCommandStatusContext(req.Context(), commandID)
	if err != nil {
		return http.StatusServiceUnavailable,
			fmt.Errorf("getting command status for ID %d: %w", commandID, err)
	}

	return http.StatusOK, output
}

// @Description  Fetches all Language Profiles from Sonarr.
// @Summary      Get Sonarr Language Profiles
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      201  {object} apps.Respond.apiResponse{message=map[int64]string} "map of ID to name"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/languageProfiles [get]
// @Security     ApiKeyAuth
func sonarrLangProfiles(req *http.Request) (int, interface{}) {
	// Get the profiles from sonarr.
	profiles, err := getSonarr(req).GetLanguageProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting language profiles: %w", err)
	}

	// Format profile ID=>Name into a nice map.
	p := make(map[int64]string)
	for i := range profiles {
		p[profiles[i].ID] = profiles[i].Name
	}

	return http.StatusOK, p
}

// @Description  Fetches all Quality Profiles Data from Sonarr.
// @Summary      Get Sonarr Quality Profile Data
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      201  {object} apps.Respond.apiResponse{message=[]sonarr.QualityProfile} "all profiles"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfile [get]
// @Security     ApiKeyAuth
func sonarrGetQualityProfile(req *http.Request) (int, interface{}) {
	// Get the profiles from sonarr.
	profiles, err := getSonarr(req).GetQualityProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting profiles: %w", err)
	}

	return http.StatusOK, profiles
}

// @Description  Fetches all Quality Profiles from Sonarr.
// @Summary      Get Sonarr Quality Profiles
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      201  {object} apps.Respond.apiResponse{message=map[int64]string} "map of ID to name"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfiles [get]
// @Security     ApiKeyAuth
func sonarrGetQualityProfiles(req *http.Request) (int, interface{}) {
	// Get the profiles from sonarr.
	profiles, err := getSonarr(req).GetQualityProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting profiles: %w", err)
	}

	// Format profile ID=>Name into a nice map.
	p := make(map[int64]string)
	for i := range profiles {
		p[profiles[i].ID] = profiles[i].Name
	}

	return http.StatusOK, p
}

// @Description  Creates a new Sonarr Quality Profile.
// @Summary      Add Sonarr Quality Profile
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.QualityProfile true "new item content"
// @Success      200  {object} apps.Respond.apiResponse{message=int64} "new profile ID"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "json input error"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfile [post]
// @Security     ApiKeyAuth
func sonarrAddQualityProfile(req *http.Request) (int, interface{}) {
	var profile sonarr.QualityProfile

	// Extract payload and check for TMDB ID.
	err := json.NewDecoder(req.Body).Decode(&profile)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	// Get the profiles from sonarr.
	id, err := getSonarr(req).AddQualityProfileContext(req.Context(), &profile)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("adding profile: %w", err)
	}

	return http.StatusOK, id
}

// @Description  Updates a Sonarr Quality Profile.
// @Summary      Update Sonarr Quality Profile
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        profileID  path   int64  true  "profile ID to update"
// @Param        PUT body sonarr.QualityProfile true "updated item content"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "json input error"
// @Failure      422  {object} apps.Respond.apiResponse{message=string} "no profile ID"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfile/{profileID} [put]
// @Security     ApiKeyAuth
func sonarrUpdateQualityProfile(req *http.Request) (int, interface{}) {
	var profile sonarr.QualityProfile

	// Extract payload and check for TMDB ID.
	err := json.NewDecoder(req.Body).Decode(&profile)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	profile.ID, _ = strconv.ParseInt(mux.Vars(req)["profileID"], mnd.Base10, mnd.Bits64)
	if profile.ID == 0 {
		return http.StatusUnprocessableEntity, ErrNonZeroID
	}

	// Get the profiles from sonarr.
	_, err = getSonarr(req).UpdateQualityProfileContext(req.Context(), &profile)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("updating profile: %w", err)
	}

	return http.StatusOK, "OK"
}

// @Description  Removes a Sonarr Quality Profile.
// @Summary      Remove Sonarr Quality Profile
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        profileID  path   int64  true  "profile ID to update"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "no profile ID"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfile/{profileID} [delete]
// @Security     ApiKeyAuth
func sonarrDeleteQualityProfile(req *http.Request) (int, interface{}) {
	profileID, _ := strconv.ParseInt(mux.Vars(req)["profileID"], mnd.Base10, mnd.Bits64)
	if profileID == 0 {
		return http.StatusBadRequest, ErrNonZeroID
	}

	// Delete the profile from sonarr.
	err := getSonarr(req).DeleteQualityProfileContext(req.Context(), profileID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("deleting profile: %w", err)
	}

	return http.StatusOK, "OK"
}

// @Description  Removes all Sonarr Quality Profiles.
// @Summary      Remove Sonarr Quality Profiles
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=apps.deleteResponse} "delete status"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error getting profiles"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualityProfiles/all [delete]
// @Security     ApiKeyAuth
func sonarrDeleteAllQualityProfiles(req *http.Request) (int, interface{}) {
	// Get all the profiles from sonarr.
	profiles, err := getSonarr(req).GetQualityProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting profiles: %w", err)
	}

	var (
		deleted int
		errs    []string
	)

	// Delete each profile from sonarr.
	for _, profile := range profiles {
		if err := getSonarr(req).DeleteQualityProfileContext(req.Context(), profile.ID); err != nil {
			errs = append(errs, err.Error())
			continue
		}

		deleted++
	}

	return http.StatusOK, &deleteResponse{
		Found:   len(profiles),
		Deleted: deleted,
		Errors:  errs,
	}
}

// @Description  Fetches all Release Profile Data from Sonarr.
// @Summary      Get Sonarr Release Profile Data
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      201  {object} apps.Respond.apiResponse{message=[]sonarr.ReleaseProfile} "all profiles"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/releaseProfile [get]
// @Security     ApiKeyAuth
func sonarrGetReleaseProfiles(req *http.Request) (int, interface{}) {
	// Get the profiles from sonarr.
	profiles, err := getSonarr(req).GetReleaseProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting profiles: %w", err)
	}

	return http.StatusOK, profiles
}

// @Description  Creates a new Sonarr Release Profile.
// @Summary      Add Sonarr Release Profile
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.ReleaseProfile true "new item content"
// @Success      200  {object} apps.Respond.apiResponse{message=int64} "new profile ID"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "json input error"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/releaseProfile [post]
// @Security     ApiKeyAuth
func sonarrAddReleaseProfile(req *http.Request) (int, interface{}) {
	var profile sonarr.ReleaseProfile

	// Extract payload and check for TMDB ID.
	err := json.NewDecoder(req.Body).Decode(&profile)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	// Get the profiles from sonarr.
	id, err := getSonarr(req).AddReleaseProfileContext(req.Context(), &profile)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("adding profile: %w", err)
	}

	return http.StatusOK, id
}

// @Description  Updates a Sonarr Release Profile.
// @Summary      Update Sonarr Release Profile
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        profileID  path   int64  true  "profile ID to update"
// @Param        PUT body sonarr.ReleaseProfile true "updated item content"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "json input error"
// @Failure      422  {object} apps.Respond.apiResponse{message=string} "no profile ID"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/releaseProfile/{profileID} [put]
// @Security     ApiKeyAuth
func sonarrUpdateReleaseProfile(req *http.Request) (int, interface{}) {
	var profile sonarr.ReleaseProfile

	// Extract payload and check for TMDB ID.
	err := json.NewDecoder(req.Body).Decode(&profile)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	profile.ID, _ = strconv.ParseInt(mux.Vars(req)["profileID"], mnd.Base10, mnd.Bits64)
	if profile.ID == 0 {
		return http.StatusUnprocessableEntity, ErrNonZeroID
	}

	// Get the profiles from sonarr.
	_, err = getSonarr(req).UpdateReleaseProfileContext(req.Context(), &profile)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("updating profile: %w", err)
	}

	return http.StatusOK, "OK"
}

// @Description  Removes a Sonarr Release Profile.
// @Summary      Remove Sonarr Release Profile
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        profileID  path   int64  true  "profile ID to update"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "no profile ID"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/releaseProfile/{profileID} [delete]
// @Security     ApiKeyAuth
func sonarrDeleteReleaseProfile(req *http.Request) (int, interface{}) {
	profileID, _ := strconv.ParseInt(mux.Vars(req)["profileID"], mnd.Base10, mnd.Bits64)
	if profileID == 0 {
		return http.StatusBadRequest, ErrNonZeroID
	}

	// Delete the profile from sonarr.
	err := getSonarr(req).DeleteReleaseProfileContext(req.Context(), profileID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("deleting profile: %w", err)
	}

	return http.StatusOK, "OK"
}

// @Description  Removes all Sonarr Release Profiles.
// @Summary      Remove Sonarr Release Profiles
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=apps.deleteResponse} "delete status"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error getting profiles"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/releaseProfile/all [delete]
// @Security     ApiKeyAuth
func sonarrDeleteAllReleaseProfiles(req *http.Request) (int, interface{}) {
	profiles, err := getSonarr(req).GetReleaseProfilesContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting profiles: %w", err)
	}

	var (
		deleted int
		errs    []string
	)

	for _, profile := range profiles {
		// Delete the profile from sonarr.
		err := getSonarr(req).DeleteReleaseProfileContext(req.Context(), profile.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		deleted++
	}

	return http.StatusOK, &deleteResponse{
		Found:   len(profiles),
		Deleted: deleted,
		Errors:  errs,
	}
}

// @Description  Returns all Sonarr Root Folders paths and free space.
// @Summary      Retrieve Sonarr Root Folders
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=map[string]int64} "map of path->space free"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/rootFolder [get]
// @Security     ApiKeyAuth
func sonarrRootFolders(req *http.Request) (int, interface{}) {
	// Get folder list from Sonarr.
	folders, err := getSonarr(req).GetRootFoldersContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting folders: %w", err)
	}

	// Format folder list into a nice path=>freesSpace map.
	p := make(map[string]int64)
	for i := range folders {
		p[folders[i].Path] = folders[i].FreeSpace
	}

	return http.StatusOK, p
}

// @Description  Returns Sonarr series naming conventions.
// @Summary      Retrieve Sonarr Series Naming
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=sonarr.Naming} "naming conventions"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/naming [get]
// @Security     ApiKeyAuth
func sonarrGetNaming(req *http.Request) (int, interface{}) {
	naming, err := getSonarr(req).GetNamingContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting naming: %w", err)
	}

	return http.StatusOK, naming
}

// @Description  Updates the Sonarr series naming conventions.
// @Summary      Update Sonarr Series Naming
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        PUT body sonarr.Naming  true  "naming conventions"
// @Success      200  {object} apps.Respond.apiResponse{message=int64} "naming ID"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "bad json input"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/naming [put]
// @Security     ApiKeyAuth
func sonarrUpdateNaming(req *http.Request) (int, interface{}) {
	var naming sonarr.Naming

	err := json.NewDecoder(req.Body).Decode(&naming)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	output, err := getSonarr(req).UpdateNamingContext(req.Context(), &naming)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("updating naming: %w", err)
	}

	return http.StatusOK, output.ID
}

// @Description  Searches all Sonarr Series Titles for the search term provided. Returns a minimal amount of data for each found item.
// @Summary      Search for Sonarr Series
// @Tags         Sonarr
// @Produce      json
// @Param        query     path   string  true  "title search string"
// @Param        instance  path   int64   true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=[]apps.sonarrSearchSeries.seriesData}  "minimal series data"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/search/{query} [get]
// @Security     ApiKeyAuth
//
//nolint:lll
func sonarrSearchSeries(req *http.Request) (int, interface{}) {
	// Get all movies
	series, err := getSonarr(req).GetAllSeriesContext(req.Context())
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("getting series: %w", err)
	}

	type seriesData struct {
		ID                int64            `json:"id"`
		Title             string           `json:"title"`
		First             time.Time        `json:"first"`
		Next              time.Time        `json:"next"`
		Previous          time.Time        `json:"prev"`
		Added             time.Time        `json:"added"`
		Status            string           `json:"status"`
		Path              string           `json:"path"`
		TvDBID            int64            `json:"tvdbId"`
		Monitored         bool             `json:"monitored"`
		QualityProfileID  int64            `json:"qualityId"`
		SeasonFolder      bool             `json:"seasonFolder"`
		SeriesType        string           `json:"seriesType"`
		LanguageProfileID int64            `json:"languageProfileId"`
		Seasons           []*sonarr.Season `json:"seasons"`
		Exists            bool             `json:"exists"`
	}

	query := strings.TrimSpace(mux.Vars(req)["query"]) // in
	resp := make([]*seriesData, 0)                     // out

	for _, item := range series {
		if seriesSearch(query, item.Title, item.AlternateTitles) {
			resp = append(resp, &seriesData{
				ID:                item.ID,
				Title:             item.Title,
				First:             item.FirstAired,
				Next:              item.NextAiring,
				Previous:          item.PreviousAiring,
				Added:             item.Added,
				Status:            item.Status,
				Path:              item.Path,
				TvDBID:            item.TvdbID,
				Monitored:         item.Monitored,
				QualityProfileID:  item.QualityProfileID,
				SeasonFolder:      item.SeasonFolder,
				SeriesType:        item.SeriesType,
				LanguageProfileID: item.LanguageProfileID,
				Seasons:           item.Seasons,
				Exists:            item.Statistics != nil && item.Statistics.SizeOnDisk > 0,
			})
		}
	}

	return http.StatusOK, resp
}

func seriesSearch(query, title string, alts []*sonarr.AlternateTitle) bool {
	if strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
		return true
	}

	for _, t := range alts {
		if strings.Contains(strings.ToLower(t.Title), strings.ToLower(query)) {
			return true
		}
	}

	return false
}

// @Description  Returns all Sonarr Tags.
// @Summary      Retrieve Sonarr Tags
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=[]starr.Tag} "tags"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/tag [get]
// @Security     ApiKeyAuth
func sonarrGetTags(req *http.Request) (int, interface{}) {
	tags, err := getSonarr(req).GetTagsContext(req.Context())
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("getting tags: %w", err)
	}

	return http.StatusOK, tags
}

// @Description  Updates the label for an existing Sonarr tag.
// @Summary      Update Sonarr Tag Label
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        tagID     path   int64  true  "tag ID to update"
// @Param        label     path   string  true  "new label"
// @Success      200  {object} apps.Respond.apiResponse{message=int64}  "tag ID"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/tag/{tagID}/{label} [put]
// @Security     ApiKeyAuth
func sonarrUpdateTag(req *http.Request) (int, interface{}) {
	id, _ := strconv.Atoi(mux.Vars(req)["tid"])

	tag, err := getSonarr(req).UpdateTagContext(req.Context(), &starr.Tag{ID: id, Label: mux.Vars(req)["label"]})
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("updating tag: %w", err)
	}

	return http.StatusOK, tag.ID
}

// @Description  Create a brand new tag in Sonarr.
// @Summary      Create Sonarr Tag
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        label     path   string  true  "tag label"
// @Success      200  {object} apps.Respond.apiResponse{message=int64}  "tag ID"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/tag/{label} [put]
// @Security     ApiKeyAuth
func sonarrSetTag(req *http.Request) (int, interface{}) {
	tag, err := getSonarr(req).AddTagContext(req.Context(), &starr.Tag{Label: mux.Vars(req)["label"]})
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("setting tag: %w", err)
	}

	return http.StatusOK, tag.ID
}

// @Description  Updates a series in Sonarr.
// @Summary      Update Sonarr Series
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path  int64  true  "instance ID"
// @Param        moveFiles query int64  true  "move files? true/false"
// @Param        PUT body sonarr.Series true  "series content"
// @Success      200  {object} apps.Respond.apiResponse{message=string}  "OK"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance} [put]
// @Security     ApiKeyAuth
func sonarrUpdateSeries(req *http.Request) (int, interface{}) {
	var series sonarr.AddSeriesInput

	err := json.NewDecoder(req.Body).Decode(&series)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	moveFiles := mux.Vars(req)["moveFiles"] == fmt.Sprint(true)

	_, err = getSonarr(req).UpdateSeriesContext(req.Context(), &series, moveFiles)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("updating series: %w", err)
	}

	return http.StatusOK, "sonarr seems to have worked"
}

// @Description  Season Pass allows you to mass-edit items in Sonarr.
// @Summary      Publish Sonarr Season Pass
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64       true  "instance ID"
// @Param        POST body sonarr.SeasonPass  true  "Season pass content"
// @Success      200  {object} apps.Respond.apiResponse{message=string}  "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "invalid json provided"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/seasonPass [post]
// @Security     ApiKeyAuth
func sonarrSeasonPass(req *http.Request) (int, interface{}) {
	var seasonPass sonarr.SeasonPass

	err := json.NewDecoder(req.Body).Decode(&seasonPass)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	err = getSonarr(req).UpdateSeasonPassContext(req.Context(), &seasonPass)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("updating seasonPass: %w", err)
	}

	return http.StatusOK, "ok"
}

// @Description  Creates a new Custom Format in Sonarr.
// @Summary      Create Sonarr Custom Format
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.CustomFormat  true  "New Custom Format content"
// @Success      200  {object} apps.Respond.apiResponse{message=sonarr.CustomFormat}  "custom format"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "invalid json provided"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/customformats [post]
// @Security     ApiKeyAuth
func sonarrAddCustomFormat(req *http.Request) (int, interface{}) {
	var cusform sonarr.CustomFormat

	err := json.NewDecoder(req.Body).Decode(&cusform)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	resp, err := getSonarr(req).AddCustomFormatContext(req.Context(), &cusform)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("adding custom format: %w", err)
	}

	return http.StatusOK, resp
}

// @Description  Returns all Custom Format from Sonarr.
// @Summary      Get Sonarr Custom Formats
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=[]sonarr.CustomFormat}  "custom formats"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/customformats [get]
// @Security     ApiKeyAuth
func sonarrGetCustomFormats(req *http.Request) (int, interface{}) {
	cusform, err := getSonarr(req).GetCustomFormatsContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting custom formats: %w", err)
	}

	return http.StatusOK, cusform
}

// @Description  Updates a Custom Format in Sonarr.
// @Summary      Update Sonarr Custom Format
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        formatID  path   int64  true  "Custom Format ID"
// @Param        PUT body sonarr.CustomFormat  true  "Updated Custom Format content"
// @Success      200  {object} apps.Respond.apiResponse{message=sonarr.CustomFormat}  "custom format"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "invalid json provided"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/customformats/{formatID} [put]
// @Security     ApiKeyAuth
func sonarrUpdateCustomFormat(req *http.Request) (int, interface{}) {
	var cusform sonarr.CustomFormat
	if err := json.NewDecoder(req.Body).Decode(&cusform); err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	cfID, _ := strconv.Atoi(mux.Vars(req)["cfid"])

	output, err := getSonarr(req).UpdateCustomFormatContext(req.Context(), &cusform, cfID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("updating custom format: %w", err)
	}

	return http.StatusOK, output
}

// @Description  Delete a Custom Format from Sonarr.
// @Summary      Delete Sonarr Custom Format
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Param        formatID  path   int64  true  "Custom Format ID"
// @Success      200  {object} apps.Respond.apiResponse{message=string}  "ok"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/customformats/{formatID} [delete]
// @Security     ApiKeyAuth
func sonarrDeleteCustomFormat(req *http.Request) (int, interface{}) {
	cfID, _ := strconv.Atoi(mux.Vars(req)["cfid"])

	err := getSonarr(req).DeleteCustomFormatContext(req.Context(), cfID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("deleting custom format: %w", err)
	}

	return http.StatusOK, "OK"
}

// @Description  Delete all Custom Formats from Sonarr.
// @Summary      Delete all Sonarr Custom Formats
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=apps.deleteResponse}  "item delete counters"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/customformats/all [delete]
// @Security     ApiKeyAuth
func sonarrDeleteAllCustomFormats(req *http.Request) (int, interface{}) {
	formats, err := getSonarr(req).GetCustomFormatsContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting custom formats: %w", err)
	}

	var (
		deleted int
		errs    []string
	)

	for _, format := range formats {
		err := getSonarr(req).DeleteCustomFormatContext(req.Context(), format.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		deleted++
	}

	return http.StatusOK, &deleteResponse{
		Found:   len(formats),
		Deleted: deleted,
		Errors:  errs,
	}
}

// @Description  Returns all Quality Definitions from Sonarr.
// @Summary      Get Sonarr Quality Definitions
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=[]sonarr.QualityDefinition}  "quality definitions list"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualitydefinition [get]
// @Security     ApiKeyAuth
func sonarrGetQualityDefinitions(req *http.Request) (int, interface{}) {
	output, err := getSonarr(req).GetQualityDefinitionsContext(req.Context())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("getting quality definitions: %w", err)
	}

	return http.StatusOK, output
}

// @Description  Updates all Quality Definitions in Sonarr.
// @Summary      Update Sonarr Quality Definitions
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        PUT body []sonarr.QualityDefinition  true  "Updated Import Listcontent"
// @Success      200  {object} apps.Respond.apiResponse{message=[]sonarr.QualityDefinition}  "quality definitions return"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "invalid json provided"
// @Failure      500  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/qualitydefinition [put]
// @Security     ApiKeyAuth
//
//nolint:lll
func sonarrUpdateQualityDefinition(req *http.Request) (int, interface{}) {
	var input []*sonarr.QualityDefinition
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	output, err := getSonarr(req).UpdateQualityDefinitionsContext(req.Context(), input)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("updating quality definition: %w", err)
	}

	return http.StatusOK, output
}

// @Description  Returns Sonarr Notifications with a name that matches 'notifiar'.
// @Summary      Retrieve Sonarr Notifications
// @Tags         Sonarr
// @Produce      json
// @Param        instance  path   int64  true  "instance ID"
// @Success      200  {object} apps.Respond.apiResponse{message=[]sonarr.NotificationOutput} "notifications"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/notifications [get]
// @Security     ApiKeyAuth
func sonarrGetNotifications(req *http.Request) (int, interface{}) {
	notifs, err := getSonarr(req).GetNotificationsContext(req.Context())
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("getting notifications: %w", err)
	}

	output := []*sonarr.NotificationOutput{}

	for _, notif := range notifs {
		if strings.Contains(strings.ToLower(notif.Name), "notifiar") {
			output = append(output, notif)
		}
	}

	return http.StatusOK, output
}

// @Description  Updates a Notifcation in Sonarr.
// @Summary      Update Sonarr Notification
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        PUT body sonarr.NotificationInput  true  "notification content"
// @Success      200  {object} apps.Respond.apiResponse{message=string} "ok"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "bad json input"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/notification [put]
// @Security     ApiKeyAuth
func sonarrUpdateNotification(req *http.Request) (int, interface{}) {
	var notif sonarr.NotificationInput

	err := json.NewDecoder(req.Body).Decode(&notif)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	_, err = getSonarr(req).UpdateNotificationContext(req.Context(), &notif)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("updating notification: %w", err)
	}

	return http.StatusOK, mnd.Success
}

// @Description  Creates a new Sonarr Notification.
// @Summary      Add Sonarr Notification
// @Tags         Sonarr
// @Produce      json
// @Accept       json
// @Param        instance  path   int64  true  "instance ID"
// @Param        POST body sonarr.NotificationInput true "new item content"
// @Success      200  {object} apps.Respond.apiResponse{message=int64} "new notification ID"
// @Failure      400  {object} apps.Respond.apiResponse{message=string} "json input error"
// @Failure      503  {object} apps.Respond.apiResponse{message=string} "instance error"
// @Failure      404  {object} string "bad token or api key"
// @Router       /api/sonarr/{instance}/notification [post]
// @Security     ApiKeyAuth
func sonarrAddNotification(req *http.Request) (int, interface{}) {
	var notif sonarr.NotificationInput

	err := json.NewDecoder(req.Body).Decode(&notif)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("decoding payload: %w", err)
	}

	id, err := getSonarr(req).AddNotificationContext(req.Context(), &notif)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("adding notification: %w", err)
	}

	return http.StatusOK, id
}
