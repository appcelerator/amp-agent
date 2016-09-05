// +build solaris

package main

import "github.com/codegangsta/cli"

var (
	checkpointCommand cli.Command
	eventsCommand     cli.Command
	restoreCommand    cli.Command
	specCommand       cli.Command
	killCommand       cli.Command
	deleteCommand     cli.Command
	execCommand       cli.Command
	initCommand       cli.Command
	listCommand       cli.Command
	pauseCommand      cli.Command
	resumeCommand     cli.Command
	startCommand      cli.Command
	stateCommand      cli.Command
)
