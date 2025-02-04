package clientinfo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"github.com/Notifiarr/notifiarr/pkg/snapshot"
	"github.com/Notifiarr/notifiarr/pkg/triggers/data"
	"github.com/Notifiarr/notifiarr/pkg/website"
	"golift.io/cnfg"
)

// ClientInfo is the client's startup data received from the website.
type ClientInfo struct {
	User struct {
		WelcomeMSG string  `json:"welcome"`
		Subscriber bool    `json:"subscriber"`
		Patron     bool    `json:"patron"`
		DevAllowed bool    `json:"devAllowed"`
		DateFormat PHPDate `json:"dateFormat"`
		StopLogs   bool    `json:"stopLogs"`
	} `json:"user"`
	Actions struct {
		Poll      bool             `json:"poll"`
		Plex      PlexConfig       `json:"plex"`      // Site Config for Plex.
		Apps      AllAppConfigs    `json:"apps"`      // Site Config for Starr.
		Dashboard DashConfig       `json:"dashboard"` // Site Config for Dashboard.
		Sync      SyncConfig       `json:"sync"`      // Site Config for TRaSH Sync.
		Gaps      GapsConfig       `json:"gaps"`      // Site Config for Radarr Gaps.
		Custom    []*CronConfig    `json:"custom"`    // Site config for Custom Crons.
		Snapshot  *snapshot.Config `json:"snapshot"`  // Site Config for System Snapshot.
	} `json:"actions"`
}

// CronConfig defines a custom GET timer from the website.
// Used to offload crons to clients.
type CronConfig struct {
	Name     string        `json:"name"`     // name of action.
	Interval cnfg.Duration `json:"interval"` // how often to GET this URI.
	URI      string        `json:"endpoint"` // endpoint for the URI.
	Desc     string        `json:"description"`
}

// SyncConfig is the configuration returned from the notifiarr website for CF/RP TraSH sync.
type SyncConfig struct {
	Interval        cnfg.Duration `json:"interval"`        // how often to fire in minutes.
	Radarr          int64         `json:"radarr"`          // deprecated: items in sync
	RadarrInstances IntList       `json:"radarrInstances"` // which instance IDs we sync
	Sonarr          int64         `json:"sonarr"`          // deprecated: items in sync
	SonarrInstances IntList       `json:"sonarrInstances"` // which instance IDs we sync
	SonarrSync      []string      `json:"sonarrSync"`      // items in sync.
	RadarrSync      []string      `json:"radarrSync"`      // items in sync.
}

// DashConfig is the configuration returned from the notifiarr website for the dashboard configuration.
type DashConfig struct {
	Interval cnfg.Duration `json:"interval"` // how often to fire in minutes.
}

// AppConfig is the data that comes from the website for each Starr app.
type AppConfig struct {
	Instance int           `json:"instance"`
	Name     string        `json:"name"`
	Corrupt  string        `json:"corrupt"`
	Backup   string        `json:"backup"`
	Interval cnfg.Duration `json:"interval"`
	Stuck    bool          `json:"stuck"`
	Finished bool          `json:"finished"`
}

// InstanceConfig allows binding methods to a list of instance configurations.
type InstanceConfig []*AppConfig

// AllAppConfigs is the configuration returned from the notifiarr website for Starr apps.
type AllAppConfigs struct {
	Lidarr   InstanceConfig `json:"lidarr"`
	Prowlarr InstanceConfig `json:"prowlarr"`
	Radarr   InstanceConfig `json:"radarr"`
	Readarr  InstanceConfig `json:"readarr"`
	Sonarr   InstanceConfig `json:"sonarr"`
}

// PlexConfig is the website-derived configuration for Plex.
type PlexConfig struct {
	Interval   cnfg.Duration `json:"interval"`
	TrackSess  bool          `json:"trackSessions"`
	AccountMap string        `json:"accountMap"`
	NoActivity bool          `json:"noActivity"`
	Delay      cnfg.Duration `json:"activityDelay"`
	Cooldown   cnfg.Duration `json:"cooldown"`
	SeriesPC   uint          `json:"seriesPc"`
	MoviesPC   uint          `json:"moviesPc"`
}

// GapsConfig is the configuration returned from the notifiarr website for Radarr Collection Gaps.
type GapsConfig struct {
	Instances IntList       `json:"instances"`
	Interval  cnfg.Duration `json:"interval"`
}

// IntList has a method to abstract lookups.
type IntList []int

// Has returns true if the list has an instance ID.
func (l IntList) Has(instance int) bool {
	for _, i := range l {
		if instance == i {
			return true
		}
	}

	return false
}

// String returns the message text for a client info response.
func (c *ClientInfo) String() string {
	if c == nil {
		return "<nil>"
	}

	return c.User.WelcomeMSG
}

// IsSub returns true if the client is a subscriber. False otherwise.
func (c *ClientInfo) IsSub() bool {
	return c != nil && c.User.Subscriber
}

// IsPatron returns true if the client is a patron. False otherwise.
func (c *ClientInfo) IsPatron() bool {
	return c != nil && c.User.Patron
}

// SaveClientInfo returns an error if the API key is wrong. Caches and returns client info otherwise.
func (c *Config) SaveClientInfo(ctx context.Context, startup bool) (*ClientInfo, error) {
	body, err := c.GetData(&website.Request{
		Route:      website.ClientRoute,
		Event:      website.EventStart,
		Payload:    c.Info(ctx, startup),
		LogPayload: true,
	})
	if err != nil {
		return nil, fmt.Errorf("sending client info: %w", err)
	}

	clientInfo := ClientInfo{}
	if err = json.Unmarshal(body.Details.Response, &clientInfo); err != nil {
		return &clientInfo, fmt.Errorf("parsing response: %w, %s", err, string(body.Details.Response))
	}

	// Only set this if there was no error.
	data.Save("clientInfo", &clientInfo)

	return &clientInfo, nil
}

func Get() *ClientInfo {
	data := data.Get("clientInfo")
	if data == nil || data.Data == nil {
		return nil
	}

	cinfo, _ := data.Data.(*ClientInfo)

	return cinfo
}

func (i InstanceConfig) Finished(instance int) bool {
	for _, app := range i {
		if app.Instance == instance {
			return app.Finished
		}
	}

	return false
}

func (i InstanceConfig) Stuck(instance int) bool {
	for _, app := range i {
		if app.Instance == instance {
			return app.Stuck
		}
	}

	return false
}

func (i InstanceConfig) Backup(instance int) string {
	for _, app := range i {
		if app.Instance == instance {
			return app.Backup
		}
	}

	return mnd.Disabled
}

func (i InstanceConfig) Corrupt(instance int) string {
	for _, app := range i {
		if app.Instance == instance {
			return app.Corrupt
		}
	}

	return mnd.Disabled
}
