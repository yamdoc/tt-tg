package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"time"
	internal "tt-tg/internal/tt-tg"

	"github.com/heilkit/tg"
)

var (
	api            = flag.String("api-tg", getApi(), "telegram api url")
	tiktokAPI      = flag.String("api-tt", getTiktok(), "tiktok/tikwm API url")
	config         = flag.String("cfg", "config.yaml", "config path")
	debug          = flag.Bool("debug", false, "print debug info")
	pollRate       = flag.Int("poll", 24, "polling rate (in hours)")
	pollRateMinute = flag.Int("poll-m", 0, "polling rate (in minutes, overrides -rate of non-zero)")
)

func main() {
	flag.Parse()

	if pollRateMinute == nil || *pollRateMinute == 0 {
		pollRateMinute = pollRate
		*pollRateMinute *= 60
	}

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
		slog.SetLogLoggerLevel(logLevel)
	}

	manager, err := internal.NewManagerFromFile(*config, *api, *tiktokAPI, logLevel)
	if err != nil {
		log.Fatalf("cannot create tt-tg manager: %v", err)
	}

	manager.Start(time.Duration(*pollRateMinute) * time.Minute)
}

func getApi() string {
	apiUrl := os.Getenv("TG_LOCAL_API")
	if apiUrl == "" {
		return tg.DefaultApiURL
	}
	return apiUrl
}

func getTiktok() string {
	apiUrl := os.Getenv("TIKWM_PROXY")
	if apiUrl == "" {
		return tg.DefaultApiURL
	}
	return apiUrl
}
