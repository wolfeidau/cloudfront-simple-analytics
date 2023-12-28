package server

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/labstack/echo/v4"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/flags"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/server/assets"
	middleware "github.com/wolfeidau/echo-middleware"
)

func Setup(cfg flags.API, awscfg aws.Config, e *echo.Echo) error {
	srv := NewAsset(cfg)

	e.GET("/c3p0/hit.png", srv.Analytics, middleware.NoCache())

	return nil
}

type Asset struct {
	cfg flags.API
}

func NewAsset(cfg flags.API) *Asset {
	return &Asset{
		cfg: cfg,
	}
}

func (gha *Asset) Analytics(c echo.Context) error {
	data, err := assets.Content.ReadFile("hit.png")
	if err != nil {
		return err
	}

	return c.Blob(http.StatusOK, "image/png", data)
}
