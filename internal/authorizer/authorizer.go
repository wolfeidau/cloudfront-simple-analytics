package authorizer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wolfeidau/cloudfront-simple-analytics/internal/flags"
)

type TokenRequest struct{}
type TokenResponse struct {
	IsAuthorized bool           `json:"isAuthorized"`
	Context      map[string]any `json:"context"`
}

type SSMToken struct {
	ssmclient *ssm.Client
	cfg       flags.Authorizer
}

func NewSSMToken(awscfg aws.Config, cfg flags.Authorizer) *SSMToken {
	return &SSMToken{
		ssm.NewFromConfig(awscfg),
		cfg,
	}
}

func (s *SSMToken) Handle(ctx context.Context, in TokenRequest) (*TokenResponse, error) {
	tr := new(TokenResponse)

	tr.IsAuthorized = true
	tr.Context = map[string]any{
		"user": "admin",
	}

	return tr, nil
}
