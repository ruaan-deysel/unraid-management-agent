package main

import (
	"log"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/cskr/pubsub"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ruaandeysel/unraid-management-agent/daemon/cmd"
	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
)

var Version string = "dev"

var cli struct {
	LogsDir string `default:"/var/log" help:"directory to store logs"`
	Port    int    `default:"8080" help:"HTTP server port"`
	Mock    bool   `env:"MOCK_MODE" default:"false" help:"enable mock mode for development"`

	Boot cmd.Boot `cmd:"" default:"1" help:"start the management agent"`
}

func main() {
	ctx := kong.Parse(&cli)

	// Set up logging with rotation
	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(cli.LogsDir, "unraid-management-agent.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 10,
		MaxAge:     28, // days
	})

	log.Printf("Starting Unraid Management Agent v%s", Version)

	// Create application context
	appCtx := &domain.Context{
		Config: domain.Config{
			Version:  Version,
			Port:     cli.Port,
			MockMode: cli.Mock,
		},
		Hub: pubsub.New(1024), // Buffer size for event bus
	}

	// Run the boot command
	err := ctx.Run(appCtx)
	ctx.FatalIfErrorf(err)
}
