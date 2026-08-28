package remote

import (
	"context"
	"net/http"

	identitysdk "github.com/domainry/domainry-identity-sdk"
)

type identityServiceTokenSource struct{ factory identitysdk.Factory }

func NewIdentityServiceTokenSource(factory identitysdk.Factory) ServiceTokenSource {
	return &identityServiceTokenSource{factory: factory}
}

func (source *identityServiceTokenSource) Token(ctx context.Context, application identitysdk.ApplicationRef) (ServiceToken, error) {
	if source == nil || source.factory == nil {
		return ServiceToken{}, remoteError(http.StatusInternalServerError, "notification.identity_service_token_source_unavailable", false, nil)
	}
	binding, err := source.factory.Open(ctx, application)
	if err != nil {
		return ServiceToken{}, remoteError(http.StatusServiceUnavailable, "notification.identity_service_token_exchange_failed", true, err)
	}
	defer binding.Close(context.Background())
	serviceBinding, ok := binding.(identitysdk.ApplicationServiceBinding)
	if !ok || serviceBinding.ApplicationServices() == nil {
		return ServiceToken{}, remoteError(http.StatusBadGateway, "notification.identity_service_token_capability_missing", false, nil)
	}
	token, err := serviceBinding.ApplicationServices().Exchange(ctx, identitysdk.ExchangeApplicationServiceTokenRequest{
		Audience: "domainry-notification",
		Grants:   []identitysdk.ApplicationServiceGrant{{Resource: "notification_event", Action: "publish"}},
	})
	if err != nil {
		return ServiceToken{}, remoteError(http.StatusServiceUnavailable, "notification.identity_service_token_exchange_failed", true, err)
	}
	return ServiceToken{AccessToken: token.AccessToken, ExpiresAt: token.ExpiresAt}, nil
}
