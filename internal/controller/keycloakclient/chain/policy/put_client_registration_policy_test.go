package policy

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func newFakeClient(t *testing.T, keycloakClient *keycloakApiAlpha.KeycloakClient) client.Client {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, keycloakApiAlpha.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	return fake.NewClientBuilder().
		WithScheme(s).
		WithStatusSubresource(&keycloakApiAlpha.KeycloakClient{}).
		WithObjects(keycloakClient).
		Build()
}

func testKeycloakClient(policies *[]keycloakApiAlpha.ClientAllowedPolicy) *keycloakApiAlpha.KeycloakClient {
	return &keycloakApiAlpha.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-client",
			Namespace: "default",
		},
		Spec: keycloakApiAlpha.KeycloakClientSpec{
			Client: []keycloakApiAlpha.Client{
				{ClientId: "test-client-id"},
			},
			ClientRegistrationPolicy: policies,
		},
	}
}

func conditionTrue() *metav1.Condition {
	return &metav1.Condition{
		Type:   chain.ConditionClientRegistrationPolicySynced,
		Status: metav1.ConditionTrue,
		Reason: chain.ReasonClientRegistrationPolicySynced,
	}
}

func TestPutClientRegistrationPolicy_Serve(t *testing.T) {
	tests := []struct {
		name              string
		policies          []keycloakApiAlpha.ClientAllowedPolicy
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
		wantCondition     *metav1.Condition
	}{
		{
			name: "config is passed through and subType is defaulted to authenticated",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a", "scope-b"},
						"allow-default-scopes":                      {"true"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-a", "scope-b"},
							"allow-default-scopes":                      {"true"},
						},
					}).Return(nil)

				return m
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name: "explicit name and subType are passed through",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					Name:       "Allowed Client Scopes",
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					SubType:    adapter.AllowedClientScopesPolicySubTypeAnonymous,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						Name:       "Allowed Client Scopes",
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAnonymous,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
						},
					}).Return(nil)

				return m
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			// providerId is required by the CRD, an empty one must not be silently substituted
			name: "providerId is passed through as is",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: "",
					Config:     map[string][]string{"some-key": {"value"}},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: "",
						SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
						Config:     map[string][]string{"some-key": {"value"}},
					}).Return(errors.New("keycloak rejected the policy"))

				return m
			},
			wantErr: require.Error,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRegistrationPolicySynced,
				Status: metav1.ConditionFalse,
				Reason: chain.ReasonKeycloakAPIError,
			},
		},
		{
			name: "every entry of the list is applied",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
					},
				},
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					SubType:    adapter.AllowedClientScopesPolicySubTypeAnonymous,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-b"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
						},
					}).Return(nil)
				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAnonymous,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-b"},
						},
					}).Return(nil)

				return m
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name: "duplicated and empty values are skipped",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a", "", "scope-a", "scope-b"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-a", "scope-b"},
						},
					}).Return(nil)

				return m
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name: "blank keys and keys left without values are dropped",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
						"":          {"value"},
						"empty-key": {""},
						"nil-key":   nil,
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm",
					&adapter.ClientRegistrationPolicy{
						ProviderID: adapter.AllowedClientScopesPolicyProviderID,
						SubType:    adapter.AllowedClientScopesPolicySubTypeAuthenticated,
						Config: map[string][]string{
							adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
						},
					}).Return(nil)

				return m
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name:     "keycloak is not called for an empty list",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				return mocks.NewMockClient(t)
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name: "entries without usable config are skipped",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{ProviderID: adapter.AllowedClientScopesPolicyProviderID, Config: nil},
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					Config:     map[string][]string{"key": {""}},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				return mocks.NewMockClient(t)
			},
			wantErr:       require.NoError,
			wantCondition: conditionTrue(),
		},
		{
			name: "error when AddClientRegistrationPolicyConfig fails",
			policies: []keycloakApiAlpha.ClientAllowedPolicy{
				{
					ProviderID: adapter.AllowedClientScopesPolicyProviderID,
					Config: map[string][]string{
						adapter.AllowedClientScopesPolicyProviderID: {"scope-a"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("AddClientRegistrationPolicyConfig", mock.Anything, "realm", mock.Anything).
					Return(errors.New("keycloak is unhappy"))

				return m
			},
			wantErr: require.Error,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientRegistrationPolicySynced,
				Status: metav1.ConditionFalse,
				Reason: chain.ReasonKeycloakAPIError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := testKeycloakClient(&tt.policies)
			objClient := newFakeClient(t, cl)

			el := NewPutClientRegistrationPolicy(tt.keycloakApiClient(t), objClient)

			dp := &DataPolicy{
				CRD:        cl,
				Policies:   tt.policies,
				Generation: cl.Generation,
			}

			err := el.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), dp, "realm")
			tt.wantErr(t, err)

			if tt.wantCondition != nil {
				cond := meta.FindStatusCondition(cl.Status.Conditions, tt.wantCondition.Type)
				require.NotNil(t, cond, "condition not found")
				require.Equal(t, tt.wantCondition.Status, cond.Status)
				require.Equal(t, tt.wantCondition.Reason, cond.Reason)
				require.Equal(t, cl.Generation, cond.ObservedGeneration)
			}
		})
	}
}
