package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/authorizer"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/flags"
	"github.com/wolfeidau/lambda-go-extras/standard"
)

var (
	version = "dev"
	cfg     flags.Authorizer
)

func main() {

	kong.Parse(&cfg,
		kong.Vars{"version": version}, // bind a var for version
	)

	awscfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load aws configuration")
	}

	ts := authorizer.NewSSMToken(awscfg, cfg)

	standard.GenericDefault(ts.Handle)
}
