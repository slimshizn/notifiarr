package client

/*
  This file contains the procedures that validate config data and initialize each app.
  All startup logs come from below. Every procedure in this file is run once on startup.
*/

import (
	"context"
	"path"

	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"github.com/Notifiarr/notifiarr/pkg/website/clientinfo"
	"golift.io/cnfg"
	"golift.io/version"
)

// PrintStartupInfo prints info about our startup config.
// This runs once on startup, and again during reloads.
func (c *Client) PrintStartupInfo(ctx context.Context, clientInfo *clientinfo.ClientInfo) {
	if clientInfo != nil {
		c.Printf("==> %s", clientInfo)
		c.printVersionChangeInfo(ctx)
	} else {
		clientInfo = &clientinfo.ClientInfo{}
	}

	switch hi, err := c.website.GetHostInfo(ctx); {
	case err != nil:
		c.Errorf("=> Unknown Host Info (this is bad): %v", err)
	case c.Config.HostID == "":
		c.Config.HostID = hi.HostID
		c.Printf("==> {UNSAVED} Unique Host ID: %s (%s)", c.Config.HostID, hi.Hostname)
	default:
		c.Printf("==> Unique Host ID: %s (%s)", hi.HostID, hi.Hostname)
	}

	c.Printf("==> %s <==", mnd.HelpLink)
	c.Printf("==> Startup Settings <==")
	c.printLidarr(&clientInfo.Actions.Apps.Lidarr)
	c.printProwlarr(&clientInfo.Actions.Apps.Prowlarr)
	c.printRadarr(&clientInfo.Actions.Apps.Radarr)
	c.printReadarr(&clientInfo.Actions.Apps.Readarr)
	c.printSonarr(&clientInfo.Actions.Apps.Sonarr)
	c.printDeluge()
	c.printNZBGet()
	c.printQbit()
	c.printRtorrent()
	c.printSABnzbd()
	c.printPlex()
	c.printTautulli()
	c.printMySQL()
	c.Printf(" => Timeout: %s, Quiet: %v", c.Config.Timeout, c.Config.Quiet)
	c.Printf(" => Trusted Upstream Networks: %v", c.Config.Allow)

	if c.Config.SSLCrtFile != "" && c.Config.SSLKeyFile != "" {
		c.Print(" => Web HTTPS Listen:", "https://"+c.Config.BindAddr+path.Join("/", c.Config.URLBase))
		c.Print(" => Web Cert & Key Files:", c.Config.SSLCrtFile+", "+c.Config.SSLKeyFile)
	} else {
		c.Print(" => Web HTTP Listen:", "http://"+c.Config.BindAddr+path.Join("/", c.Config.URLBase))
	}

	c.printLogFileInfo()
}

func (c *Client) printVersionChangeInfo(ctx context.Context) {
	const clientVersion = "clientVersion"

	values, err := c.website.GetState(ctx, clientVersion)
	if err != nil {
		c.Errorf("XX> Getting version from database: %v", err)
	}

	previousVersion := string(values[clientVersion])
	if previousVersion == version.Version ||
		version.Version == "" {
		return
	}

	c.Printf("==> Detected application version change! %s => %s", previousVersion, version.Version)

	err = c.website.SetState(ctx, clientVersion, []byte(version.Version))
	if err != nil {
		c.Errorf("Updating version in database: %v", err)
	}
}

func (c *Client) printLogFileInfo() { //nolint:cyclop
	if c.Config.LogFile != "" {
		if c.Config.LogFiles > 0 {
			c.Printf(" => Log File: %s (%d @ %dMb)", c.Config.LogFile, c.Config.LogFiles, c.Config.LogFileMb)
		} else {
			c.Printf(" => Log File: %s (no rotation)", c.Config.LogFile)
		}
	}

	if c.Config.HTTPLog != "" {
		if c.Config.LogFiles > 0 {
			c.Printf(" => HTTP Log: %s (%d @ %dMb)", c.Config.HTTPLog, c.Config.LogFiles, c.Config.LogFileMb)
		} else {
			c.Printf(" => HTTP Log: %s (no rotation)", c.Config.HTTPLog)
		}
	}

	if c.Config.Debug && c.Config.LogConfig.DebugLog != "" {
		if c.Config.LogFiles > 0 {
			c.Printf(" => Debug Log: %s (%d @ %dMb)", c.Config.LogConfig.DebugLog, c.Config.LogFiles, c.Config.LogFileMb)
		} else {
			c.Printf(" => Debug Log: %s (no rotation)", c.Config.LogConfig.DebugLog)
		}
	}

	if c.Config.Services.LogFile != "" && !c.Config.Services.Disabled && len(c.Config.Service) > 0 {
		if c.Config.LogFiles > 0 {
			c.Printf(" => Service Checks Log: %s (%d @ %dMb)", c.Config.Services.LogFile, c.Config.LogFiles, c.Config.LogFileMb)
		} else {
			c.Printf(" => Service Checks Log: %s (no rotation)", c.Config.Services.LogFile)
		}
	}
}

// printPlex is called on startup to print info about configured Plex instance(s).
func (c *Client) printPlex() {
	plex := c.Config.Plex
	if !plex.Enabled() {
		return
	}

	name := plex.Server.Name()
	if name == "" {
		name = "<connection error?>"
	}

	c.Printf(" => Plex Config: 1 server: %s @ %s (enables incoming APIs and webhook) timeout:%v check_interval:%s ",
		name, plex.URL, plex.Timeout, plex.Interval)
}

// printLidarr is called on startup to print info about each configured server.
func (c *Client) printLidarr(app *clientinfo.InstanceConfig) {
	s := "s"
	if len(c.Config.Lidarr) == 1 {
		s = ""
	}

	c.Print(" => Lidarr Config:", len(c.Config.Lidarr), "server"+s)

	for idx, f := range c.Config.Lidarr {
		c.Printf(" =>    Server %d: %s apikey:%v timeout:%s valid_ssl:%v stuck/fin:%v/%v corrupt:%v backup:%v "+
			"http/pass:%v/%v",
			idx+1, f.URL, f.APIKey != "", f.Timeout, f.ValidSSL, app.Stuck(idx+1), app.Finished(idx+1),
			app.Corrupt(idx+1) != "" && app.Corrupt(idx+1) != mnd.Disabled, app.Backup(idx+1) != mnd.Disabled,
			f.HTTPPass != "" && f.HTTPUser != "", f.Password != "" && f.Username != "")
	}
}

// printProwlarr is called on startup to print info about each configured server.
func (c *Client) printProwlarr(app *clientinfo.InstanceConfig) {
	s := "s"
	if len(c.Config.Prowlarr) == 1 {
		s = ""
	}

	c.Print(" => Prowlarr Config:", len(c.Config.Prowlarr), "server"+s)

	for idx, f := range c.Config.Prowlarr {
		c.Printf(" =>    Server %d: %s apikey:%v timeout:%s valid_ssl:%v corrupt:%v backup:%v "+
			"http/pass:%v/%v",
			idx+1, f.URL, f.APIKey != "", f.Timeout, f.ValidSSL,
			app.Corrupt(idx+1) != "" && app.Corrupt(idx+1) != mnd.Disabled, app.Backup(idx+1) != mnd.Disabled,
			f.HTTPPass != "" && f.HTTPUser != "", f.Password != "" && f.Username != "")
	}
}

// printRadarr is called on startup to print info about each configured server.
func (c *Client) printRadarr(app *clientinfo.InstanceConfig) {
	s := "s"
	if len(c.Config.Radarr) == 1 {
		s = ""
	}

	c.Print(" => Radarr Config:", len(c.Config.Radarr), "server"+s)

	for idx, f := range c.Config.Radarr {
		c.Printf(" =>    Server %d: %s apikey:%v timeout:%s valid_ssl:%v stuck/fin:%v/%v corrupt:%v backup:%v "+
			"http/pass:%v/%v",
			idx+1, f.URL, f.APIKey != "", f.Timeout, f.ValidSSL, app.Stuck(idx+1), app.Finished(idx+1),
			app.Corrupt(idx+1) != "" && app.Corrupt(idx+1) != mnd.Disabled, app.Backup(idx+1) != mnd.Disabled,
			f.HTTPPass != "" && f.HTTPUser != "", f.Password != "" && f.Username != "")
	}
}

// printReadarr is called on startup to print info about each configured server.
func (c *Client) printReadarr(app *clientinfo.InstanceConfig) {
	s := "s"
	if len(c.Config.Readarr) == 1 {
		s = ""
	}

	c.Print(" => Readarr Config:", len(c.Config.Readarr), "server"+s)

	for idx, f := range c.Config.Readarr {
		c.Printf(" =>    Server %d: %s apikey:%v timeout:%s valid_ssl:%v stuck/fin:%v/%v corrupt:%v backup:%v "+
			"http/pass:%v/%v",
			idx+1, f.URL, f.APIKey != "", f.Timeout, f.ValidSSL, app.Stuck(idx+1), app.Finished(idx+1),
			app.Corrupt(idx+1) != "" && app.Corrupt(idx+1) != mnd.Disabled, app.Backup(idx+1) != mnd.Disabled,
			f.HTTPPass != "" && f.HTTPUser != "", f.Password != "" && f.Username != "")
	}
}

// printSonarr is called on startup to print info about each configured server.
func (c *Client) printSonarr(app *clientinfo.InstanceConfig) {
	s := "s"
	if len(c.Config.Sonarr) == 1 {
		s = ""
	}

	c.Print(" => Sonarr Config:", len(c.Config.Sonarr), "server"+s)

	for idx, f := range c.Config.Sonarr {
		c.Printf(" =>    Server %d: %s apikey:%v timeout:%s valid_ssl:%v stuck/fin:%v/%v corrupt:%v backup:%v "+
			"http/pass:%v/%v",
			idx+1, f.URL, f.APIKey != "", f.Timeout, f.ValidSSL, app.Stuck(idx+1), app.Finished(idx+1),
			app.Corrupt(idx+1) != "" && app.Corrupt(idx+1) != mnd.Disabled, app.Backup(idx+1) != mnd.Disabled,
			f.HTTPPass != "" && f.HTTPUser != "", f.Password != "" && f.Username != "")
	}
}

// printDeluge is called on startup to print info about each configured server.
func (c *Client) printDeluge() {
	s := "s"
	if len(c.Config.Deluge) == 1 {
		s = ""
	}

	c.Print(" => Deluge Config:", len(c.Config.Deluge), "server"+s)

	for i, f := range c.Config.Deluge {
		c.Printf(" =>    Server %d: %s password:%v timeout:%s valid_ssl:%v",
			i+1, f.Config.URL, f.Password != "", cnfg.Duration{Duration: f.Timeout.Duration}, f.ValidSSL)
	}
}

// printNZBGet is called on startup to print info about each configured server.
func (c *Client) printNZBGet() {
	s := "s"
	if len(c.Config.NZBGet) == 1 {
		s = ""
	}

	c.Print(" => NZBGet Config:", len(c.Config.NZBGet), "server"+s)

	for i, f := range c.Config.NZBGet {
		c.Printf(" =>    Server %d: %s username:%s password:%v timeout:%s valid_ssl:%v",
			i+1, f.Config.URL, f.User, f.Pass != "", cnfg.Duration{Duration: f.Timeout.Duration}, f.ValidSSL)
	}
}

// printQbit is called on startup to print info about each configured server.
func (c *Client) printQbit() {
	s := "s"
	if len(c.Config.Qbit) == 1 {
		s = ""
	}

	c.Print(" => Qbit Config:", len(c.Config.Qbit), "server"+s)

	for i, f := range c.Config.Qbit {
		c.Printf(" =>    Server %d: %s username:%s password:%v timeout:%s valid_ssl:%v",
			i+1, f.Config.URL, f.User, f.Pass != "", cnfg.Duration{Duration: f.Timeout.Duration}, f.ValidSSL)
	}
}

// printRtorrent is called on startup to print info about each configured server.
func (c *Client) printRtorrent() {
	s := "s"
	if len(c.Config.Rtorrent) == 1 {
		s = ""
	}

	c.Print(" => rTorrent Config:", len(c.Config.Rtorrent), "server"+s)

	for i, f := range c.Config.Rtorrent {
		c.Printf(" =>    Server %d: %s username:%s password:%v timeout:%s valid_ssl:%v",
			i+1, f.URL, f.User, f.Pass != "", cnfg.Duration{Duration: f.Timeout.Duration}, f.ValidSSL)
	}
}

// printSABnzbd is called on startup to print info about each configured SAB downloader.
func (c *Client) printSABnzbd() {
	s := "s"
	if len(c.Config.SabNZB) == 1 {
		s = ""
	}

	c.Print(" => SABnzbd Config:", len(c.Config.SabNZB), "server"+s)

	for i, f := range c.Config.SabNZB {
		c.Printf(" =>    Server %d: %s, api_key:%v timeout:%s", i+1, f.URL, f.APIKey != "", f.Timeout)
	}
}

// printTautulli is called on startup to print info about configured Tautulli instance(s).
func (c *Client) printTautulli() {
	switch taut := c.Config.Apps.Tautulli; {
	case !taut.Enabled():
		c.Printf(" => Tautulli Config (enables name map): 0 servers")
	case taut.Name != "":
		c.Printf(" => Tautulli Config (enables name map): 1 server: %s timeout:%v check_interval:%s name:%s",
			taut.URL, taut.Timeout, taut.Interval, taut.Name)
	default:
		c.Printf(" => Tautulli Config (enables name map): 1 server: %s timeout:%s", taut.URL, taut.Timeout)
	}
}

// printMySQL is called on startup to print info about each configured SQL server.
func (c *Client) printMySQL() {
	if c.Config.Snapshot.Plugins == nil { // unlikely.
		return
	}

	s := "s"
	if len(c.Config.Snapshot.MySQL) == 1 {
		s = ""
	}

	c.Print(" => MySQL Config:", len(c.Config.Snapshot.MySQL), "server"+s)

	for i, m := range c.Config.Snapshot.MySQL {
		if m.Name != "" {
			c.Printf(" =>    Server %d: %s user:%v timeout:%s check_interval:%s name:%s",
				i+1, m.Host, m.User, m.Timeout, m.Interval, m.Name)
		} else {
			c.Printf(" =>    Server %d: %s user:%v timeout:%s", i+1, m.Host, m.User, m.Timeout)
		}
	}
}
