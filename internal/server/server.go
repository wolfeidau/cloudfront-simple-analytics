package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/apex/gateway/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/flags"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/server/assets"
	middleware "github.com/wolfeidau/echo-middleware"
)

func Setup(cfg flags.API, awscfg aws.Config, e *echo.Echo) error {
	srv := NewAsset(cfg, awscfg)

	e.GET("/c3p1/hit.png", srv.Analytics, middleware.NoCache())

	return nil
}

type HitRecord struct {
	CloudfrontForwardedProto          string `header:"cloudfront-forwarded-proto" json:"cf_forwarded_proto,omitempty"`
	CloudfrontIsAndroidViewer         string `header:"cloudfront-is-android-viewer" json:"cf_is_android_vwr,omitempty"`
	CloudfrontIsDesktopViewer         string `header:"cloudfront-is-desktop-viewer" json:"cf_is_desktop_vwr,omitempty"`
	CloudfrontIsIosViewer             string `header:"cloudfront-is-ios-viewer" json:"cf_is_ios_vwr,omitempty"`
	CloudfrontIsMobileViewer          string `header:"cloudfront-is-mobile-viewer" json:"cf_is_mobile_vwr,omitempty"`
	CloudfrontIsSmarttvViewer         string `header:"cloudfront-is-smarttv-viewer" json:"cf_is_smarttv_vwr,omitempty"`
	CloudfrontIsTabletViewer          string `header:"cloudfront-is-tablet-viewer" json:"cf_is_tablet_vwr,omitempty"`
	CloudfrontViewerAddress           string `header:"cloudfront-viewer-address" json:"cf_vwr_address,omitempty"`
	CloudfrontViewerAsn               string `header:"cloudfront-viewer-asn" json:"cf_vwr_asn,omitempty"`
	CloudfrontViewerCity              string `header:"cloudfront-viewer-city" json:"cf_vwr_city,omitempty"`
	CloudfrontViewerCountry           string `header:"cloudfront-viewer-country" json:"cf_vwr_country,omitempty"`
	CloudfrontViewerCountryName       string `header:"cloudfront-viewer-country-name" json:"cf_vwr_country_name,omitempty"`
	CloudfrontViewerCountryRegion     string `header:"cloudfront-viewer-country-region" json:"cf_vwr_country_region,omitempty"`
	CloudfrontViewerCountryRegionName string `header:"cloudfront-viewer-country-region-name" json:"cf_vwr_country_region_name,omitempty"`
	CloudfrontViewerHttpVersion       string `header:"cloudfront-viewer-http-version" json:"cf_vwr_http_version,omitempty"`
	CloudfrontViewerJa3Fingerprint    string `header:"cloudfront-viewer-ja3-fingerprint" json:"cf_vwr_ja3_fingerprint,omitempty"`
	CloudfrontViewerLatitude          string `header:"cloudfront-viewer-latitude" json:"cf_vwr_latitude,omitempty"`
	CloudfrontViewerLongitude         string `header:"cloudfront-viewer-longitude" json:"cf_vwr_longitude,omitempty"`
	CloudfrontViewerPostalCode        string `header:"cloudfront-viewer-postal-code" json:"cf_vwr_postal_code,omitempty"`
	CloudfrontViewerTimeZone          string `header:"cloudfront-viewer-time-zone" json:"cf_vwr_time_zone,omitempty"`
	CloudfrontViewerTls               string `header:"cloudfront-viewer-tls" json:"cf_vwr_tls,omitempty"`
	Host                              string `header:"host" json:"host,omitempty"`
	Referer                           string `header:"referer" json:"referer,omitempty"`
	UserAgent                         string `header:"user-agent" json:"user_agent,omitempty"`
	Via                               string `header:"via" json:"via,omitempty"`
	XAmzCfId                          string `header:"x-amz-cf-id" json:"x_amz_cf_id,omitempty"`
	XAmznTraceId                      string `header:"x-amzn-trace-id" json:"x_amzn_trace_id,omitempty"`
	TS                                string `header:"-" json:"ts,omitempty"`
	TSEpochMillis                     int64  `header:"-" json:"ts_epoch_millis,omitempty"`
	UTMId                             string `header:"-" json:"utm_id,omitempty"`
	UTMSource                         string `header:"-" json:"utm_source,omitempty"`
	UTMMedium                         string `header:"-" json:"utm_medium,omitempty"`
	UTMCampaign                       string `header:"-" json:"utm_campaign,omitempty"`
	UTMTerm                           string `header:"-" json:"utm_term,omitempty"`
	UTMContent                        string `header:"-" json:"utm_content,omitempty"`
}

// updateTS use the lambda request context to set the timestamp to the current time if we have one
// as this is set by the lambda runtime and is probably more accurate than time in the lambda container
// fall back to local time if not available
func (hr *HitRecord) updateTS(ctx context.Context) {

	if requestContext, ok := gateway.RequestContext(ctx); ok {
		hr.TS = time.Unix(0, requestContext.TimeEpoch*int64(time.Millisecond)).Format(time.RFC3339)
		hr.TSEpochMillis = requestContext.TimeEpoch

		return // we are done
	}

	hr.TS = time.Now().Format(time.RFC3339)
	hr.TSEpochMillis = time.Now().UnixNano() / int64(time.Millisecond)
}

func (hr *HitRecord) updateCampaignURL(referer string) error {
	if referer == "" {
		return nil
	}

	url, err := url.Parse(referer)
	if err != nil {
		return err
	}

	// utm_source and utm_medium are required
	if url.Query().Get("utm_source") == "" && url.Query().Get("utm_medium") == "" {
		return fmt.Errorf("referer missing utm_source and utm_medium: %s", referer)
	}

	hr.UTMId = url.Query().Get("utm_id")
	hr.UTMSource = url.Query().Get("utm_source")
	hr.UTMMedium = url.Query().Get("utm_medium")
	hr.UTMCampaign = url.Query().Get("utm_campaign")
	hr.UTMTerm = url.Query().Get("utm_term")
	hr.UTMContent = url.Query().Get("utm_content")

	return nil
}

type Asset struct {
	cfg      flags.API
	firehose *firehose.Client
}

func NewAsset(cfg flags.API, awscfg aws.Config) *Asset {
	return &Asset{
		firehose: firehose.NewFromConfig(awscfg),
		cfg:      cfg,
	}
}

func (gha *Asset) Analytics(c echo.Context) error {
	data, err := assets.Content.ReadFile("hit.png")
	if err != nil {
		return err
	}

	request := new(HitRecord)
	binder := new(echo.DefaultBinder)

	err = binder.BindHeaders(c, request)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	err = request.updateCampaignURL(c.Request().Header.Get("referer"))
	if err != nil {
		// TODO: change to a metric
		log.Ctx(ctx).Warn().Err(err).Msg("failed to parse campaign url")
	}

	request.updateTS(ctx)

	log.Ctx(c.Request().Context()).Info().Any("request", request).Msg("hit")

	recordData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	res, err := gha.firehose.PutRecord(ctx, &firehose.PutRecordInput{
		DeliveryStreamName: aws.String(gha.cfg.DeliverStreamName),
		Record: &types.Record{
			Data: recordData,
		},
	})
	if err != nil {
		return err
	}

	log.Ctx(c.Request().Context()).Info().Any("request", request).Str("record_id", aws.ToString(res.RecordId)).Msg("hit")

	return c.Blob(http.StatusOK, "image/png", data)
}
