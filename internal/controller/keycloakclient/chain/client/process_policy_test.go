package client

import (
	"context"
	"errors"
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
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestProcessPolicy_Serve(t *testing.T) {
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
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "client authorization is not set",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				return mocks.NewMockClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "policies processed successfully",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "aggregate-policy",
										Type:        keycloakApiAlpha.PolicyTypeAggregate,
										Description: "Aggregate policy",
										AggregatedPolicy: &keycloakApiAlpha.AggregatedPolicyData{
											Policies: []string{"role-policy", "user-policy"},
										},
									},
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        keycloakApiAlpha.PolicyTypeClient,
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
									{
										Name:        "group-policy",
										Description: "Group policy",
										Type:        keycloakApiAlpha.PolicyTypeGroup,
										GroupPolicy: &keycloakApiAlpha.GroupPolicyData{
											Groups: []keycloakApiAlpha.GroupDefinition{
												{
													Name: "test-group",
												},
											},
										},
									},
									{
										Name:        "role-policy",
										Description: "Role policy",
										Type:        keycloakApiAlpha.PolicyTypeRole,
										RolePolicy: &keycloakApiAlpha.RolePolicyData{
											Roles: []keycloakApiAlpha.RoleDefinition{
												{
													Name:     "test-role",
													Required: true,
												},
											},
										},
									},
									{
										Name:        "time-policy",
										Description: "Time policy",
										Type:        keycloakApiAlpha.PolicyTypeTime,
										TimePolicy: &keycloakApiAlpha.TimePolicyData{
											NotBefore:    "2024-03-03 00:00:00",
											NotOnOrAfter: "2024-03-03 00:00:00",
										},
									},
									{
										Name:        "user-policy",
										Description: "User policy",
										Type:        keycloakApiAlpha.PolicyTypeUser,
										UserPolicy: &keycloakApiAlpha.UserPolicyData{
											Users: []string{"test-user"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"Default Policy": {
							ID: gocloak.StringP("default-policy-id"),
						},
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"role-policy": {
							ID: gocloak.StringP("role-policy-id"),
						},
						"user-policy2": {
							ID: gocloak.StringP("user-policy2-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("GetGroups", mock.Anything, "master").
					Return(map[string]*gocloak.Group{
						"test-group": {
							ID: gocloak.StringP("test-group-id"),
						},
					}, nil)
				m.On("GetRealmRoles", mock.Anything, "master").
					Return(map[string]gocloak.Role{
						"test-role": {
							ID: gocloak.StringP("test-role-id"),
						},
					}, nil)
				m.On("GetUsersByNames", mock.Anything, "master", []string{"test-user"}).
					Return(map[string]gocloak.User{
						"test-user": {
							ID: gocloak.StringP("test-user-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(&gocloak.PolicyRepresentation{}, nil).Times(4)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(2)
				m.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy2-id").
					Return(nil).Once()

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "policies addOnly successful",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ReconciliationStrategy: keycloakApiAlpha.ReconciliationStrategyAddOnly,
							ClientId:               "test-client-2",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "aggregate-policy",
										Type:        keycloakApiAlpha.PolicyTypeAggregate,
										Description: "Aggregate policy",
										AggregatedPolicy: &keycloakApiAlpha.AggregatedPolicyData{
											Policies: []string{"role-policy", "user-policy"},
										},
									},
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        keycloakApiAlpha.PolicyTypeClient,
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client-2"},
										},
									},
									{
										Name:        "group-policy",
										Description: "Group policy",
										Type:        keycloakApiAlpha.PolicyTypeGroup,
										GroupPolicy: &keycloakApiAlpha.GroupPolicyData{
											Groups: []keycloakApiAlpha.GroupDefinition{
												{
													Name: "test-group",
												},
											},
										},
									},
									{
										Name:        "role-policy",
										Description: "Role policy",
										Type:        keycloakApiAlpha.PolicyTypeRole,
										RolePolicy: &keycloakApiAlpha.RolePolicyData{
											Roles: []keycloakApiAlpha.RoleDefinition{
												{
													Name:     "test-role",
													Required: true,
												},
											},
										},
									},
									{
										Name:        "time-policy",
										Description: "Time policy",
										Type:        keycloakApiAlpha.PolicyTypeTime,
										TimePolicy: &keycloakApiAlpha.TimePolicyData{
											NotBefore:    "2024-03-03 00:00:00",
											NotOnOrAfter: "2024-03-03 00:00:00",
										},
									},
									{
										Name:        "user-policy",
										Description: "User policy",
										Type:        keycloakApiAlpha.PolicyTypeUser,
										UserPolicy: &keycloakApiAlpha.UserPolicyData{
											Users: []string{"test-user"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client-2", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"Default Policy": {
							ID: gocloak.StringP("default-policy-id"),
						},
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"role-policy": {
							ID: gocloak.StringP("role-policy-id"),
						},
						"user-policy2": {
							ID: gocloak.StringP("user-policy2-id"),
						},
						"existing-policy": {
							ID: gocloak.StringP("existing-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client-2": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("GetGroups", mock.Anything, "master").
					Return(map[string]*gocloak.Group{
						"test-group": {
							ID: gocloak.StringP("test-group-id"),
						},
					}, nil)
				m.On("GetRealmRoles", mock.Anything, "master").
					Return(map[string]gocloak.Role{
						"test-role": {
							ID: gocloak.StringP("test-role-id"),
						},
					}, nil)
				m.On("GetUsersByNames", mock.Anything, "master", []string{"test-user"}).
					Return(map[string]gocloak.User{
						"test-user": {
							ID: gocloak.StringP("test-user-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(&gocloak.PolicyRepresentation{}, nil).Times(4)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(2)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to delete policy",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        keycloakApiAlpha.PolicyTypeClient,
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"client-policy": {
							ID: gocloak.StringP("client-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(1)
				m.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy-id").
					Return(errors.New("failed to delete policy")).Once()

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete policy")
			},
		},
		{
			name: "failed to update policy",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        keycloakApiAlpha.PolicyTypeClient,
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"client-policy": {
							ID: gocloak.StringP("client-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(errors.New("failed to update policy")).Times(1)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update policy")
			},
		},
		{
			name: "failed to create policy",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        keycloakApiAlpha.PolicyTypeClient,
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil, errors.New("failed to crate policy")).Times(1)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create policy")
			},
		},
		{
			name: "invalid policy type",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        "invalid",
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to convert policy")
			},
		},
		{
			name: "failed to get policies",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        "invalid",
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(nil, errors.New("failed to get policies"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get policies")
			},
		},
		{
			name: "failed to get client id",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ClientId: "test-client",
							Authorization: &keycloakApiAlpha.Authorization{
								Policies: []keycloakApiAlpha.Policy{
									{
										Name:        "client-policy",
										Description: "Client policy",
										Type:        "invalid",
										ClientPolicy: &keycloakApiAlpha.ClientPolicyData{
											Clients: []string{"test-client"},
										},
									},
								},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("", errors.New("failed to get client id"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get client id")
			},
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

			h := NewProcessPolicy(tt.keycloakApiClient(t), k8sClient)

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
