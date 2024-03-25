package main

import (
	"fmt"
	"github.com/gotify/server/v2/api/stream"
	"math/rand"
	"os"
	"time"

	"github.com/gotify/server/v2/config"
	"github.com/gotify/server/v2/database"
	"github.com/gotify/server/v2/mode"
	"github.com/gotify/server/v2/model"
	"github.com/gotify/server/v2/router"
	"github.com/gotify/server/v2/runner"
	"github.com/gotify/server/v2/scheduler"
)

var (
	// Version the version of Gotify.
	Version = "unknown"
	// Commit the git commit hash of this version.
	Commit = "unknown"
	// BuildDate the date on which this binary was build.
	BuildDate = "unknown"
	// Mode the build mode.
	Mode = mode.Dev
)

func main() {
	vInfo := &model.VersionInfo{Version: Version, Commit: Commit, BuildDate: BuildDate}
	mode.Set(Mode)

	fmt.Println("Starting Gotify version", vInfo.Version+"@"+BuildDate)
	rand.Seed(time.Now().UnixNano())
	conf := config.Get()

	if conf.PluginsDir != "" {
		if err := os.MkdirAll(conf.PluginsDir, 0o755); err != nil {
			panic(err)
		}
	}
	if err := os.MkdirAll(conf.UploadedImagesDir, 0o755); err != nil {
		panic(err)
	}

	db, err := database.New(conf.Database.Dialect, conf.Database.Connection, conf.DefaultUser.Name, conf.DefaultUser.Pass, conf.PassStrength, true)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	streamHandler := stream.New(time.Duration(conf.Server.Stream.PingPeriodSeconds)*time.Second, 15*time.Second, conf.Server.Stream.AllowedOrigins)

	jobScheduler, schedulerCloseable := scheduler.Init(db, streamHandler)
	defer schedulerCloseable()

	engine, routerCloseable := router.Create(db, vInfo, conf, streamHandler, jobScheduler)
	defer routerCloseable()

	if err := runner.Run(engine, conf); err != nil {
		fmt.Println("Server error: ", err)
		os.Exit(1)
	}
}
