package client

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestProcessScope_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
	)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApiAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name              string
		keycloakClient    *keycloakApiAlpha.KeycloakClient
		keycloakApiClient func(t *testing.T) keycloak.Client
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name:           "client has no authorization settings",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "get scopes",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "client",
							Authorization: &keycloakApiAlpha.Authorization{
								Scopes: []string{"scopeID1"},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("scopeID1"),
						},
					}, nil).Once()
				client.On("CreateScope", mock.Anything, "master", "clientID", "scopeID1").Return(nil, nil).Once()
				client.On("DeleteScope", mock.Anything, "master", "clientID", "scopeID").Return(nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "scope created successfully",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "client",
							Authorization: &keycloakApiAlpha.Authorization{
								Scopes: []string{"token-exchange"},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("token-exchange"),
						},
					}, nil).Once()
				client.On(
					"CreateScope",
					mock.Anything,
					"master",
					"clientID",
					"token-exchange").
					Return(&gocloak.ScopeRepresentation{Name: gocloak.StringP("token-exchange")}, nil)
				client.On("DeleteScope", mock.Anything, "master", "clientID", "scopeID").Return(nil)

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "scope deleted successfully",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "client",
							Authorization: &keycloakApiAlpha.Authorization{
								Scopes: []string{},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("token-exchange"),
						},
					}, nil).Once()
				client.On(
					"DeleteScope",
					mock.Anything,
					"master",
					"clientID",
					"scopeID").
					Return(nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure test client has proper metadata
			if tt.keycloakClient.Name == "" {
				tt.keycloakClient.Name = testClientName
			}

			if tt.keycloakClient.Namespace == "" {
				tt.keycloakClient.Namespace = testClientNamespace
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			h := NewProcessScope(tt.keycloakApiClient(t), k8sClient)

			for index, itemClient := range tt.keycloakClient.Spec.Client {
				dc := &DataClient{
					CRD:         tt.keycloakClient,
					Client:      itemClient,
					ClientIndex: index,
					Namespace:   tt.keycloakClient.Namespace,
					ReasonScope: &ReasonScope{
						ClientID:   tt.keycloakClient.Status.GetClientIDByName("test-client"),
						Generation: tt.keycloakClient.Generation,
					},
				}

				err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), dc, "master")

				tt.wantErr(t, err)
			}
		})
	}
}
