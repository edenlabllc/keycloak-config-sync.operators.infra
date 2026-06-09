package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakmocks "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestServiceAccount_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApiAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApiAlpha.KeycloakClient
		mockSetup      func(*keycloakmocks.MockClient)
		realmName      string
		expectedError  string
	}{
		{
			name: "success - service account disabled (nil)",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							Name:           "test-client",
							ServiceAccount: nil,
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account disabled (enabled=false)",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							Name: "test-client",
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled: false,
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - service account with public client",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							Public: true,
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled: true,
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected - should fail before calling API
			},
			realmName:     "test-realm",
			expectedError: "service account can not be configured with public client",
		},
		{
			name: "success - basic service account with full reconciliation",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ReconciliationStrategy: keycloakApiAlpha.ReconciliationStrategyFull,
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1", "realm-role2"},
								ClientRoles: []keycloakApiAlpha.UserClientRole{
									{
										ClientID: "client1",
										Roles:    []string{"role1", "role2"},
									},
									{
										ClientID: "client2",
										Roles:    []string{"role3"},
									},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1", "role2"},
					"client2": {"role3"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1", "realm-role2"}, expectedClientRoles, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - basic service account with add-only reconciliation",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ReconciliationStrategy: keycloakApiAlpha.ReconciliationStrategyAddOnly,
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
								ClientRoles: []keycloakApiAlpha.UserClientRole{
									{
										ClientID: "client1",
										Roles:    []string{"role1"},
									},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, expectedClientRoles, true).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account with groups",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
								Groups:     []string{"group1", "group2"},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1", "group2"}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account with attributes",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
								AttributesV2: map[string][]string{
									"attr1": {"value1", "value2"},
									"attr2": {"value3"},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedAttributes := map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				}

				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)

				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - full service account with all features",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ReconciliationStrategy: keycloakApiAlpha.ReconciliationStrategyAddOnly,
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1", "realm-role2"},
								ClientRoles: []keycloakApiAlpha.UserClientRole{
									{
										ClientID: "client1",
										Roles:    []string{"role1", "role2"},
									},
								},
								Groups: []string{"group1", "group2"},
								AttributesV2: map[string][]string{
									"attr1": {"value1"},
									"attr2": {"value2", "value3"},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1", "role2"},
				}
				expectedAttributes := map[string][]string{
					"attr1": {"value1"},
					"attr2": {"value2", "value3"},
				}

				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1", "realm-role2"}, expectedClientRoles, true).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1", "group2"}, true).Return(nil)
				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, true).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - SyncServiceAccountRoles fails",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(errors.New("roles sync failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account roles: roles sync failed",
		},
		{
			name: "error - SyncServiceAccountGroups fails",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
								Groups:     []string{"group1"},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1"}, false).Return(errors.New("groups sync failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account groups: groups sync failed",
		},
		{
			name: "error - SetServiceAccountAttributes fails",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
								AttributesV2: map[string][]string{
									"attr1": {"value1"},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedAttributes := map[string][]string{
					"attr1": {"value1"},
				}

				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, false).Return(errors.New("attributes set failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to set service account attributes: attributes set failed",
		},
		{
			name: "success - empty client roles",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:     true,
								RealmRoles:  []string{"realm-role1"},
								ClientRoles: []keycloakApiAlpha.UserClientRole{},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - empty realm roles",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{},
								ClientRoles: []keycloakApiAlpha.UserClientRole{
									{
										ClientID: "client1",
										Roles:    []string{"role1"},
									},
								},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{}, expectedClientRoles, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - default reconciliation strategy (full)",
			keycloakClient: &keycloakApiAlpha.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakClientSpec{
					Client: []keycloakApiAlpha.Client{
						{
							// ReconciliationStrategy not specified, should default to full
							ServiceAccount: &keycloakApiAlpha.ServiceAccount{
								Enabled:    true,
								RealmRoles: []string{"realm-role1"},
							},
						},
					},
				},
				Status: keycloakApiAlpha.KeycloakClientStatus{
					ClientIDs: map[string]string{
						"test-client": "client-123",
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			// Ensure test client has proper metadata
			if tt.keycloakClient.Name == "" {
				tt.keycloakClient.Name = "test-client"
			}

			if tt.keycloakClient.Namespace == "" {
				tt.keycloakClient.Namespace = "default"
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			// Create the service
			service := NewServiceAccount(mockClient, k8sClient)

			// Execute the method
			for index, client := range tt.keycloakClient.Spec.Client {
				dc := &DataClient{
					CRD:         tt.keycloakClient,
					Client:      client,
					ClientIndex: index,
					Namespace:   tt.keycloakClient.Namespace,
					ReasonScope: &ReasonScope{
						ClientID:   tt.keycloakClient.Status.GetClientIDByName("test-client"),
						Generation: tt.keycloakClient.Generation,
					},
				}

				err := service.Serve(context.Background(), dc, tt.realmName)

				// Assert the result
				if tt.expectedError != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}
