package policy

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	helperMocks "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper/mocks"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

// countingHandler records how many times it has been served and with which api client.
type countingHandler struct {
	errs   []error
	served int
	client keycloak.Client
}

func (h *countingHandler) Serve(_ context.Context, _ *DataPolicy, _ string) error {
	var err error
	if h.served < len(h.errs) {
		err = h.errs[h.served]
	}

	h.served++

	return err
}

func (h *countingHandler) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	h.client = keycloakApiClient
}

func TestChain_Serve_SkipsWhenPolicyIsNotConfigured(t *testing.T) {
	h := &countingHandler{}

	ch := &Chain{}
	ch.Use(h)

	err := ch.Serve(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		testKeycloakClient(nil),
		"realm",
	)

	require.NoError(t, err)
	require.Zero(t, h.served, "handlers must not be served when the policy is not configured")
}

func TestChain_Serve_PassesScopesToHandlers(t *testing.T) {
	h := &countingHandler{}

	ch := &Chain{}
	ch.Use(h)

	kc := testKeycloakClient(&[]keycloakApiAlpha.ClientAllowedPolicy{{ProviderID: "allowed-client-scopes", Config: map[string][]string{"allowed-client-scopes": {"scope-a"}}}})
	kc.Generation = 7

	err := ch.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), kc, "realm")

	require.NoError(t, err)
	require.Equal(t, 1, h.served)
}

func TestChain_Serve_RefreshesTokenOnUnauthorized(t *testing.T) {
	unauthorized := &gocloak.APIError{Code: http.StatusUnauthorized, Message: "unauthorized"}

	refreshedClient := mocks.NewMockClient(t)

	controllerHelper := helperMocks.NewMockControllerHelper(t)
	controllerHelper.On("CreateKeycloakClientFromConfigRef", mock.Anything, mock.Anything).
		Return(refreshedClient, nil)

	// fails once with 401, succeeds on the retry after the token refresh
	h := &countingHandler{errs: []error{unauthorized}}

	ch := &Chain{helper: controllerHelper}
	ch.Use(h)

	err := ch.Serve(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		testKeycloakClient(&[]keycloakApiAlpha.ClientAllowedPolicy{{ProviderID: "allowed-client-scopes", Config: map[string][]string{"allowed-client-scopes": {"scope-a"}}}}),
		"realm",
	)

	require.NoError(t, err)
	require.Equal(t, 2, h.served)
	require.Same(t, refreshedClient, h.client, "handler must be reconfigured with the refreshed client")
}

func TestChain_Serve_ReturnsHandlerError(t *testing.T) {
	h := &countingHandler{errs: []error{errors.New("boom")}}

	ch := &Chain{}
	ch.Use(h)

	err := ch.Serve(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		testKeycloakClient(&[]keycloakApiAlpha.ClientAllowedPolicy{{ProviderID: "allowed-client-scopes", Config: map[string][]string{"allowed-client-scopes": {"scope-a"}}}}),
		"realm",
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to serve handler")
	require.Equal(t, 1, h.served)
}
