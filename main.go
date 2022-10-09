//nolint:godot
package main

import (
	"log"
	"runtime/debug"

	"github.com/Notifiarr/notifiarr/pkg/client"
	"github.com/Notifiarr/notifiarr/pkg/mnd"
	"github.com/Notifiarr/notifiarr/pkg/ui"
)

// @title Notifiarr Client API Documentation
// @version 1.0
// @description Monitors local services and sends notifications.
// @termsOfService https://notifiarr.com
// @contact.name Notifiarr Discord
// @contact.url https://notifiarr.com/discord
// @license.name MIT
// @license.url https://github.com/Notifiarr/notifiarr/blob/main/LICENSE
// @BasePath /
func main() {
	ui.HideConsoleWindow()
	// setup log package in case we throw an error in main.go before logging is setup.
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[ERROR] ")

	defer func() {
		if r := recover(); r != nil {
			ui.ShowConsoleWindow()
			log.Printf("Go Panic! %s\n%v\n%s", mnd.BugIssue, r, string(debug.Stack()))
		}
	}()

	if err := client.Start(); err != nil {
		_, _ = ui.Error(mnd.Title, err.Error())
		log.Fatal(err) //nolint:gocritic // defer does not need to run if we have an error.
	}
}
