package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestPutAdminFineGrainedPermissions_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "with admin permission enabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApiAlpha.KeycloakClient{}).
					WithObjects(
						&keycloakApiAlpha.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApiAlpha.KeycloakClientSpec{
								Client: []keycloakApiAlpha.Client{
									{
										ClientId:                           "test-client-id",
										AdminFineGrainedPermissionsEnabled: true,
										Permission: &keycloakApiAlpha.AdminFineGrainedPermission{
											ScopePermissions: []keycloakApiAlpha.ScopePermissions{
												{
													Name:     "map-role",
													Policies: []string{"scope permission"},
												},
											},
										},
									},
								},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				scopePermissions := map[string]string{
					"map-role": "321",
				}

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), "ADMIN_FINE_GRAINED_AUTHZ").
					Return(true, nil).
					Once()

				m.On("GetClientID", "test-client-id", "realm").
					Return("123", nil).
					Once()

				m.On("GetClientID", "realm-management", "realm").
					Return("567", nil).
					Once()

				m.On("UpdateClientManagementPermissions", "realm", "123", adapter.ManagementPermissionRepresentation{
					Enabled: gocloak.BoolP(true),
				}).
					Return(nil)

				m.On("GetClientManagementPermissions", "realm", "123").
					Return(&adapter.ManagementPermissionRepresentation{
						Enabled:          gocloak.BoolP(true),
						ScopePermissions: &scopePermissions,
					}, nil)

				m.On("GetPermissions", ctrl.LoggerInto(context.Background(), logr.Discard()), "realm", "567").
					Return(map[string]gocloak.PermissionRepresentation{
						"token-exchange": {
							ID:   gocloak.StringP("scope-permission-id"),
							Name: gocloak.StringP("scope permission"),
						},
						"map-role": {
							ID:   gocloak.StringP("scope-permission2-id"),
							Name: gocloak.StringP("scope-permission2"),
						},
					}, nil).Once()

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag disabled",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApiAlpha.KeycloakClient{}).
					WithObjects(
						&keycloakApiAlpha.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApiAlpha.KeycloakClientSpec{
								Client: []keycloakApiAlpha.Client{
									{
										ClientId:                           "test-client-id",
										AdminFineGrainedPermissionsEnabled: true,
										Permission: &keycloakApiAlpha.AdminFineGrainedPermission{
											ScopePermissions: []keycloakApiAlpha.ScopePermissions{
												{
													Name:     "map-role",
													Policies: []string{"scope permission"},
												},
											},
										},
									},
								},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), "ADMIN_FINE_GRAINED_AUTHZ").
					Return(false, nil).
					Once()

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with feature flag check error",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithStatusSubresource(&keycloakApiAlpha.KeycloakClient{}).
					WithObjects(
						&keycloakApiAlpha.KeycloakClient{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-client",
								Namespace: "default",
							},
							Spec: keycloakApiAlpha.KeycloakClientSpec{
								Client: []keycloakApiAlpha.Client{
									{
										ClientId:                           "test-client-id",
										AdminFineGrainedPermissionsEnabled: true,
										Permission: &keycloakApiAlpha.AdminFineGrainedPermission{
											ScopePermissions: []keycloakApiAlpha.ScopePermissions{
												{
													Name:     "map-role",
													Policies: []string{"scope permission"},
												},
											},
										},
									},
								},
							},
						}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("FeatureFlagEnabled", ctrl.LoggerInto(context.Background(), logr.Discard()), "ADMIN_FINE_GRAINED_AUTHZ").
					Return(false, fmt.Errorf("feature flag check failed")).
					Once()

				return m
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &keycloakApiAlpha.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			el := NewPutAdminFineGrainedPermissions(tt.keycloakApiClient(t), tt.client(t))

			for index, itemClient := range cl.Spec.Client {
				dc := &DataClient{
					CRD:         cl,
					Client:      itemClient,
					ClientIndex: index,
					Namespace:   cl.Namespace,
					ReasonScope: &ReasonScope{
						ClientID:   cl.Status.GetClientIDByName("test-client"),
						Generation: cl.Generation,
					},
				}

				err := el.Serve(
					ctrl.LoggerInto(context.Background(), logr.Discard()),
					dc,
					"realm",
				)
				tt.wantErr(t, err)
			}
		})
	}
}
