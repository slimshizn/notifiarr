//go:build darwin || windows

package client

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/Notifiarr/notifiarr/pkg/bindata"
	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"github.com/Notifiarr/notifiarr/pkg/triggers/common"
	"github.com/Notifiarr/notifiarr/pkg/ui"
	"github.com/Notifiarr/notifiarr/pkg/website"
	"github.com/Notifiarr/notifiarr/pkg/website/clientinfo"
	"github.com/getlantern/systray"
	"golift.io/starr"
	"golift.io/version"
)

/* This file handles the OS GUI elements. */

// This is arbitrary to avoid conflicts.
const timerPrefix = "TimErr"

// This variable holds all the menu items.
var menu = make(map[string]*systray.MenuItem) //nolint:gochecknoglobals

// startTray Run()s readyTray to bring up the web server and the GUI app.
func (c *Client) startTray(ctx context.Context, cancel context.CancelFunc, clientInfo *clientinfo.ClientInfo) {
	systray.Run(func() {
		defer os.Exit(0)
		defer c.CapturePanic()

		b, _ := bindata.Asset(ui.SystrayIcon)
		systray.SetTemplateIcon(b, b)
		systray.SetTooltip(c.Flags.Name() + " v" + version.Version)
		c.makeChannels() // make these before starting the web server.
		c.makeMoreChannels()
		c.setupChannels(ctx, c.watchKillerChannels, c.watchNotifiarrMenu, c.watchLogsChannels,
			c.watchConfigChannels, c.watchGuiChannels, c.watchTopChannels)
		c.setupMenus(clientInfo)

		// This starts the web server, and waits for reload/exit signals.
		if err := c.Exit(ctx, cancel); err != nil {
			c.Errorf("Server: %v", err)
			os.Exit(1) // web server problem
		}
	}, func() {
		// This code only fires from menu->quit.
		if err := c.stop(ctx, website.EventUser); err != nil {
			c.Errorf("Server: %v", err)
			os.Exit(1) // web server problem
		}
		// because systray wants to control the exit code? no..
		os.Exit(0)
	})
}

func (c *Client) setupMenus(clientInfo *clientinfo.ClientInfo) {
	if !ui.HasGUI() {
		return
	}

	if !c.Config.Debug {
		menu["debug"].Hide()
	} else {
		menu["debug"].Show()

		if c.Config.LogConfig.DebugLog == "" {
			menu["debug_logs"].Hide()
			menu["debug_logs2"].Hide()
		} else {
			menu["debug_logs"].Show()
			menu["debug_logs2"].Show()
		}
	}

	if c.Config.Services.LogFile == "" {
		menu["logs_svcs"].Hide()
	} else {
		menu["logs_svcs"].Show()
	}

	if !c.Config.Services.Disabled {
		menu["svcs"].Check()
	} else {
		menu["svcs"].Uncheck()
	}

	if clientInfo == nil {
		return
	}

	go c.buildDynamicTimerMenus()

	if clientInfo.IsSub() {
		menu["sub"].SetTitle("Subscriber \u2764\ufe0f")
		menu["sub"].Check()
		menu["sub"].Disable()
		menu["sub"].SetTooltip("THANK YOU for supporting the project!")
	} else if clientInfo.IsPatron() {
		menu["sub"].SetTitle("Patron \U0001f9e1")
		menu["sub"].SetTooltip("THANK YOU for supporting the project!")
		menu["sub"].Check()
	}
}

// setupChannels runs the channel watcher loops in go routines with a panic catcher.
func (c *Client) setupChannels(ctx context.Context, funcs ...func(ctx context.Context)) {
	for _, f := range funcs {
		go func(ctx context.Context, fn func(context.Context)) {
			defer c.CapturePanic()
			fn(ctx)
		}(ctx, f)
	}
}

func (c *Client) makeChannels() {
	menu["stat"] = systray.AddMenuItem("Running", "web server state unknown")

	conf := systray.AddMenuItem("Config", "show configuration")
	menu["conf"] = conf
	menu["view"] = conf.AddSubMenuItem("View", "show configuration")
	menu["edit"] = conf.AddSubMenuItem("Edit", "edit configuration")
	menu["pass"] = conf.AddSubMenuItem("Password", "create or update the Web UI admin password")
	menu["write"] = conf.AddSubMenuItem("Write", "write config file")
	menu["svcs"] = conf.AddSubMenuItem("Services", "toggle service checks routine")
	menu["load"] = conf.AddSubMenuItem("Reload", "reload configuration")

	link := systray.AddMenuItem("Links", "external resources")
	menu["link"] = link
	menu["info"] = link.AddSubMenuItem(c.Flags.Name(), version.Print(c.Flags.Name()))
	menu["info"].Disable()
	menu["hp"] = link.AddSubMenuItem("Notifiarr.com", "open Notifiarr.com")
	menu["wiki"] = link.AddSubMenuItem("Notifiarr.Wiki", "open Notifiarr wiki")
	menu["trash"] = link.AddSubMenuItem("TRaSH Guide", "open TRaSH wiki for Notifiarr")
	menu["disc1"] = link.AddSubMenuItem("Notifiarr Discord", "open Notifiarr discord server")
	menu["disc2"] = link.AddSubMenuItem("Go Lift Discord", "open Go Lift discord server")
	menu["gh"] = link.AddSubMenuItem("GitHub Project", c.Flags.Name()+" on GitHub")

	logs := systray.AddMenuItem("Logs", "log file info")
	menu["logs"] = logs
	menu["logs_view"] = logs.AddSubMenuItem("View", "view the application log")
	menu["logs_http"] = logs.AddSubMenuItem("HTTP", "view the HTTP log")
	menu["debug_logs2"] = logs.AddSubMenuItem("Debug", "view the Debug log")
	menu["logs_svcs"] = logs.AddSubMenuItem("Services", "view the Services log")
	menu["logs_rotate"] = logs.AddSubMenuItem("Rotate", "rotate both log files")
}

// makeMoreChannels makes the Notifiarr menu and Debug menu items.
//
//nolint:lll
func (c *Client) makeMoreChannels() {
	data := systray.AddMenuItem("Notifiarr", "plex sessions, system snapshots, service checks")
	menu["data"] = data
	menu["gaps"] = data.AddSubMenuItem("Send Radarr Gaps", "[premium feature] trigger radarr collections gaps")
	menu["synccf"] = data.AddSubMenuItem("TRaSH: Sync Radarr", "[premium feature] trigger TRaSH radarr sync")
	menu["syncqp"] = data.AddSubMenuItem("TRaSH: Sync Sonarr", "[premium feature] trigger TRaSH sonarr sync")
	menu["svcs_prod"] = data.AddSubMenuItem("Check and Send Services", "check all services and send results to notifiarr")
	menu["plex_prod"] = data.AddSubMenuItem("Send Plex Sessions", "send plex sessions to notifiarr")
	menu["snap_prod"] = data.AddSubMenuItem("Send System Snapshot", "send system snapshot to notifiarr")
	menu["send_dash"] = data.AddSubMenuItem("Send Dashboard States", "collect and send all application states for a dashboard update")
	menu["corrLidarr"] = data.AddSubMenuItem("Check Lidarr Corruption", "check latest backup database in each instance for corruption")
	menu["corrProwlarr"] = data.AddSubMenuItem("Check Prowlarr Corruption", "check latest backup database in each instance for corruption")
	menu["corrRadarr"] = data.AddSubMenuItem("Check Radarr Corruption", "check latest backup database in each instance for corruption")
	menu["corrReadarr"] = data.AddSubMenuItem("Check Readarr Corruption", "check latest backup database in each instance for corruption")
	menu["corrSonarr"] = data.AddSubMenuItem("Check Sonarr Corruption", "check latest backup database in each instance for corruption")
	menu["backLidarr"] = data.AddSubMenuItem("Send Lidarr Backups", "send backup file list for each instance to Notifiarr")
	menu["backProwlarr"] = data.AddSubMenuItem("Send Prowlarr Backups", "send backup file list for each instance to Notifiarr")
	menu["backRadarr"] = data.AddSubMenuItem("Send Radarr Backups", "send backup file list for each instance to Notifiarr")
	menu["backReadarr"] = data.AddSubMenuItem("Send Readarr Backups", "send backup file list for each instance to Notifiarr")
	menu["backSonarr"] = data.AddSubMenuItem("Send Sonarr Backups", "send backup file list for each instance to Notifiarr")
	// custom timers get added onto data after this.

	debug := systray.AddMenuItem("Debug", "Debug Menu")
	menu["debug"] = debug
	menu["debug_logs"] = debug.AddSubMenuItem("View Debug Log", "view the Debug log")
	menu["svcs_log"] = debug.AddSubMenuItem("Log Service Checks", "check all services and log results")
	menu["console"] = debug.AddSubMenuItem("Console", "toggle the console window")

	if runtime.GOOS != mnd.Windows {
		menu["console"].Hide()
	}

	debug.AddSubMenuItem("- Danger Zone -", "").Disable()
	menu["debug_panic"] = debug.AddSubMenuItem("Application Panic", "cause an application panic (crash)")
	menu["update"] = systray.AddMenuItem("Update", "check GitHub for updated version")
	menu["gui"] = systray.AddMenuItem("Open WebUI", "open the web page for this Notifiarr client")
	menu["sub"] = systray.AddMenuItem("Subscribe", "subscribe for premium features")
	menu["exit"] = systray.AddMenuItem("Quit", "exit "+c.Flags.Name())
}

// Listen to the top-menu-item channels so they don't back up with junk.
func (c *Client) watchTopChannels(ctx context.Context) {
	for {
		select {
		case <-menu["conf"].ClickedCh: // unused, top menu.
		case <-menu["link"].ClickedCh: // unused, top menu.
		case <-menu["logs"].ClickedCh: // unused, top menu.
		case <-menu["data"].ClickedCh: // unused, top menu.
		case <-menu["debug"].ClickedCh: // unused, top menu.
		}
	}
}

func (c *Client) closeDynamicTimerMenus() {
	for name := range menu {
		if !strings.HasPrefix(name, timerPrefix) || menu[name].ClickedCh == nil {
			continue
		}

		close(menu[name].ClickedCh)
		menu[name].ClickedCh = nil
	}
}

// dynamic & reusable menu items with reflection, anyone?
func (c *Client) buildDynamicTimerMenus() {
	defer c.CapturePanic()

	timers := c.triggers.CronTimer.List()
	if len(timers) == 0 {
		return
	}

	if menu["timerinfo"] == nil {
		menu["timerinfo"] = menu["data"].AddSubMenuItem("- Custom Timers -", "")
	} else {
		// Re-use the already-created menu. This happens after reload.
		menu["timerinfo"].Show()
	}

	menu["timerinfo"].Disable()
	defer menu["timerinfo"].Hide()

	cases := make([]reflect.SelectCase, len(timers))

	for idx, timer := range timers {
		desc := fmt.Sprintf("%s; config: interval: %s, path: %s", timer.Desc, timer.Interval, timer.URI)
		if timer.Desc == "" {
			desc = fmt.Sprintf("dynamic custom timer; config: interval: %s, path: %s", timer.Interval, timer.URI)
		}

		name := timerPrefix + timer.Name
		if menu[name] == nil {
			menu[name] = menu["data"].AddSubMenuItem(timer.Name, desc)
		} else {
			// Re-use the already-created menu. This happens after reload.
			menu[name].ClickedCh = make(chan struct{})
			menu[name].SetTooltip(desc)
		}

		menu[name].Show()
		defer menu[name].Hide()

		cases[idx] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(menu[name].ClickedCh)}
	}

	c.Printf("==> Created %d Notifiarr custom timer menu channels.", len(cases))
	defer c.Printf("!!> All %d Notifiarr custom timer menu channels stopped.", len(cases))

	for {
		if idx, _, ok := reflect.Select(cases); ok {
			timers[idx].Run(&common.ActionInput{Type: website.EventUser})
		} else if cases = append(cases[:idx], cases[idx+1:]...); len(cases) < 1 {
			// Channel cases[idx] has been closed, remove it.
			return // no menus left to watch, exit.
		}
	}
}

func (c *Client) watchKillerChannels(ctx context.Context) {
	defer systray.Quit() // this kills the app

	for {
		select {
		case <-menu["exit"].ClickedCh:
			c.Printf("Need help? %s\n=====> Exiting! User Requested", mnd.HelpLink)
			return
		case <-menu["debug_panic"].ClickedCh:
			c.menuPanic()
		case <-menu["load"].ClickedCh:
			c.triggerConfigReload(website.EventUser, "User Requested")
		}
	}
}

//nolint:errcheck
func (c *Client) watchGuiChannels(ctx context.Context) {
	for {
		select {
		case <-menu["stat"].ClickedCh:
			c.toggleServer(ctx)
		case <-menu["gh"].ClickedCh:
			go ui.OpenURL("https://github.com/Notifiarr/notifiarr/")
		case <-menu["hp"].ClickedCh:
			go ui.OpenURL("https://notifiarr.com/")
		case <-menu["wiki"].ClickedCh:
			go ui.OpenURL("https://notifiarr.wiki/")
		case <-menu["trash"].ClickedCh:
			go ui.OpenURL("https://trash-guides.info/Notifiarr/Quick-Start/")
		case <-menu["disc1"].ClickedCh:
			go ui.OpenURL("https://notifiarr.com/discord")
		case <-menu["disc2"].ClickedCh:
			go ui.OpenURL("https://golift.io/discord")
		case <-menu["sub"].ClickedCh:
			go ui.OpenURL("https://github.com/sponsors/Notifiarr")
		}
	}
}

//nolint:errcheck
func (c *Client) watchConfigChannels(ctx context.Context) {
	for {
		select {
		case <-menu["view"].ClickedCh:
			go ui.Info(mnd.Title+": Configuration", c.displayConfig())
		case <-menu["pass"].ClickedCh:
			c.updatePassword(ctx)
		case <-menu["edit"].ClickedCh:
			go ui.OpenFile(c.Flags.ConfigFile)
			c.Print("user requested] Editing Config File:", c.Flags.ConfigFile)
		case <-menu["write"].ClickedCh:
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			c.writeConfigFile(ctx)
			cancel()
		case <-menu["console"].ClickedCh:
			if menu["console"].Checked() {
				menu["console"].Uncheck()
				ui.HideConsoleWindow()
			} else {
				menu["console"].Check()
				ui.ShowConsoleWindow()
			}
		case <-menu["svcs"].ClickedCh:
			if menu["svcs"].Checked() {
				menu["svcs"].Uncheck()
				c.Config.Services.Stop()
				ui.Notify("Stopped checking services!")
			} else {
				menu["svcs"].Check()
				c.Config.Services.Start(ctx)
				ui.Notify("Service checks started!")
			}
		}
	}
}

//nolint:errcheck
func (c *Client) watchLogsChannels(ctx context.Context) {
	for {
		select {
		case <-menu["logs_view"].ClickedCh:
			go ui.OpenLog(c.Config.LogFile)
			c.Print("[user requested] Viewing App Log File:", c.Config.LogFile)
		case <-menu["logs_http"].ClickedCh:
			go ui.OpenLog(c.Config.HTTPLog)
			c.Print("[user requested] Viewing HTTP Log File:", c.Config.HTTPLog)
		case <-menu["logs_svcs"].ClickedCh:
			go ui.OpenLog(c.Config.Services.LogFile)
			c.Print("[user requested] Viewing Services Log File:", c.Config.Services.LogFile)
		case <-menu["debug_logs"].ClickedCh:
			go ui.OpenLog(c.Config.LogConfig.DebugLog)
			c.Print("[user requested] Viewing Debug File:", c.Config.LogConfig.DebugLog)
		case <-menu["debug_logs2"].ClickedCh:
			go ui.OpenLog(c.Config.LogConfig.DebugLog)
			c.Print("[user requested] Viewing Debug File:", c.Config.LogConfig.DebugLog)
		case <-menu["logs_rotate"].ClickedCh:
			c.rotateLogs()
		case <-menu["update"].ClickedCh:
			go c.checkForUpdate(ctx)
		case <-menu["gui"].ClickedCh:
			c.openGUI()
		}
	}
}

//nolint:errcheck,cyclop
func (c *Client) watchNotifiarrMenu(ctx context.Context) {
	for {
		select {
		case <-menu["gaps"].ClickedCh:
			c.triggers.Gaps.Send(website.EventUser)
		case <-menu["synccf"].ClickedCh:
			c.triggers.CFSync.SyncRadarrCF(website.EventUser)
		case <-menu["syncqp"].ClickedCh:
			c.triggers.CFSync.SyncSonarrRP(website.EventUser)
		case <-menu["svcs_log"].ClickedCh:
			c.Print("[user requested] Checking services and logging results.")
			ui.Notify("Running and logging %d Service Checks.", len(c.Config.Service))
			c.Config.Services.RunChecks("log")
		case <-menu["svcs_prod"].ClickedCh:
			c.Print("[user requested] Checking services and sending results to Notifiarr.")
			ui.Notify("Running and sending %d Service Checks.", len(c.Config.Service))
			c.Config.Services.RunChecks(website.EventUser)
		case <-menu["plex_prod"].ClickedCh:
			c.triggers.PlexCron.Send(website.EventUser)
		case <-menu["snap_prod"].ClickedCh:
			c.triggers.SnapCron.Send(website.EventUser)
		case <-menu["send_dash"].ClickedCh:
			c.triggers.Dashboard.Send(website.EventUser)
		case <-menu["corrLidarr"].ClickedCh:
			_ = c.triggers.Backups.Corruption(&common.ActionInput{Type: website.EventUser}, starr.Lidarr)
		case <-menu["corrProwlarr"].ClickedCh:
			_ = c.triggers.Backups.Corruption(&common.ActionInput{Type: website.EventUser}, starr.Prowlarr)
		case <-menu["corrRadarr"].ClickedCh:
			_ = c.triggers.Backups.Corruption(&common.ActionInput{Type: website.EventUser}, starr.Radarr)
		case <-menu["corrReadarr"].ClickedCh:
			_ = c.triggers.Backups.Corruption(&common.ActionInput{Type: website.EventUser}, starr.Readarr)
		case <-menu["corrSonarr"].ClickedCh:
			_ = c.triggers.Backups.Corruption(&common.ActionInput{Type: website.EventUser}, starr.Sonarr)
		case <-menu["backLidarr"].ClickedCh:
			_ = c.triggers.Backups.Backup(&common.ActionInput{Type: website.EventUser}, starr.Lidarr)
		case <-menu["backProwlarr"].ClickedCh:
			_ = c.triggers.Backups.Backup(&common.ActionInput{Type: website.EventUser}, starr.Prowlarr)
		case <-menu["backRadarr"].ClickedCh:
			_ = c.triggers.Backups.Backup(&common.ActionInput{Type: website.EventUser}, starr.Radarr)
		case <-menu["backReadarr"].ClickedCh:
			_ = c.triggers.Backups.Backup(&common.ActionInput{Type: website.EventUser}, starr.Readarr)
		case <-menu["backSonarr"].ClickedCh:
			_ = c.triggers.Backups.Backup(&common.ActionInput{Type: website.EventUser}, starr.Sonarr)
		}
	}
}
