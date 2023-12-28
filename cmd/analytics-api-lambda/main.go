package main

import (
	"context"
	"io"

	"github.com/alecthomas/kong"
	"github.com/apex/gateway/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/labstack/echo/v4"
	echolog "github.com/labstack/gommon/log"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/flags"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/server"
	"github.com/wolfeidau/lambda-go-extras/standard"
)

var (
	version = "dev"
	cfg     flags.API
)

func main() {
	kong.Parse(&cfg,
		kong.Vars{"version": version}, // bind a var for version
	)

	awscfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load aws configuration")
	}

	e := echo.New()

	// shut down all the default output of echo
	e.Logger.SetOutput(io.Discard)
	e.Logger.SetLevel(echolog.OFF)

	gw := gateway.NewGateway(e)

	err = server.Setup(cfg, awscfg, e)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup server routes")
	}

	standard.Default(gw)
}
