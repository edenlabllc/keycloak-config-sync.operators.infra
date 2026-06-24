package client

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestPutClientRole_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		realmName         string
		wantErr           require.ErrorAssertionFunc
		wantCondition     *metav1.Condition
	}{
		{
			name: "success - sync client roles with ClientRoles",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId:    "test-client-id",
										ClientRoles: []string{"role1", "role2"},
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).Return(nil)

				return m
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientRolesSynced,
			},
		},
		{
			name: "success - sync client roles with ClientRolesV2",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId: "test-client-id",
										ClientRolesV2: []keycloakApiAlpha.ClientRole{
											{
												Name:        "admin-role",
												Description: "Administrator role",
											},
											{
												Name:        "user-role",
												Description: "User role",
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).Return(nil)

				return m
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientRolesSynced,
			},
		},
		{
			name: "success - no client roles",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId: "test-client-id",
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).Return(nil)

				return m
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientRolesSynced,
			},
		},
		{
			name: "success - empty client roles arrays",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId:      "test-client-id",
										ClientRoles:   []string{},
										ClientRolesV2: []keycloakApiAlpha.ClientRole{},
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).Return(nil)

				return m
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientRolesSynced,
			},
		},
		{
			name: "error - SyncClientRoles fails",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId:    "test-client-id",
										ClientRoles: []string{"role1"},
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).
					Return(errors.New("keycloak API error"))

				return m
			},
			realmName: "test-realm",
			wantErr:   require.Error,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionFalse,
				Reason: chain.ReasonKeycloakAPIError,
			},
		},
		{
			name: "success - client roles with associated roles",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApiAlpha.AddToScheme(s))

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
										ClientId: "test-client-id",
										ClientRolesV2: []keycloakApiAlpha.ClientRole{
											{
												Name:                  "composite-role",
												Description:           "Composite role",
												AssociatedClientRoles: []string{"role1", "role2"},
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
				m.On("SyncClientRoles", mock.Anything, "test-realm", mock.Anything).Return(nil)

				return m
			},
			realmName: "test-realm",
			wantErr:   require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRolesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientRolesSynced,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &keycloakApiAlpha.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			el := NewPutClientRole(tt.keycloakApiClient(t), tt.client(t))

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
					tt.realmName,
				)
				tt.wantErr(t, err)

				// Assert condition is set correctly
				if tt.wantCondition != nil {
					wantConditionType := fmt.Sprintf("%s.%s",
						helper.RemoveSpecialChar(itemClient.Name), tt.wantCondition.Type)

					cond := meta.FindStatusCondition(cl.Status.Conditions, wantConditionType)
					require.NotNil(t, cond, "condition not found")
					require.Equal(t, tt.wantCondition.Status, cond.Status)
					require.Equal(t, tt.wantCondition.Reason, cond.Reason)
					require.Equal(t, cl.Generation, cond.ObservedGeneration)
				}
			}
		})
	}
}
