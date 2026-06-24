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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/mocks"
)

func TestPutClientScope_Serve(t *testing.T) {
	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
		wantCondition     *metav1.Condition
	}{
		{
			name: "with default scopes",
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
										ClientId:            "test-client-id",
										DefaultClientScopes: []string{"default-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
			},
			wantErr: require.NoError,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientScopesSynced,
				Status: metav1.ConditionTrue,
				Reason: chain.ReasonClientScopesSynced,
			},
		},
		{
			name: "with optional scopes",
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
										ClientId:             "test-client-id",
										OptionalClientScopes: []string{"optional-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with both default and optional scopes",
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
										ClientId:             "test-client-id",
										DefaultClientScopes:  []string{"default-scope-1", "default-scope-2"},
										OptionalClientScopes: []string{"optional-scope-1", "optional-scope-2"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"default-scope-1", "default-scope-2"}).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"optional-scope-1", "optional-scope-2"}).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with no scopes",
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

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "error when GetClientScopesByNames fails for default scopes",
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
										ClientId:            "test-client-id",
										DefaultClientScopes: []string{"default-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("failed to get client scopes"))

				return m
			},
			wantErr: require.Error,
			wantCondition: &metav1.Condition{
				Type:   chain.ConditionClientScopesSynced,
				Status: metav1.ConditionFalse,
				Reason: chain.ReasonKeycloakAPIError,
			},
		},
		{
			name: "error when GetClientScopesByNames fails for optional scopes",
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
										ClientId:             "test-client-id",
										OptionalClientScopes: []string{"optional-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("failed to get client scopes"))

				return m
			},
			wantErr: require.Error,
		},
		{
			name: "error when AddDefaultScopeToClient fails",
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
										ClientId:            "test-client-id",
										DefaultClientScopes: []string{"default-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add default scope"))

				return m
			},
			wantErr: require.Error,
		},
		{
			name: "error when AddOptionalScopeToClient fails",
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
										ClientId:             "test-client-id",
										OptionalClientScopes: []string{"optional-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add optional scope"))

				return m
			},
			wantErr: require.Error,
		},
		{
			name: "error when AddOptionalScopeToClient fails with both default and optional scopes",
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
										ClientId:             "test-client-id",
										DefaultClientScopes:  []string{"default-scope"},
										OptionalClientScopes: []string{"optional-scope"},
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

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"default-scope"}).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, []string{"optional-scope"}).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("failed to add optional scope"))

				return m
			},
			wantErr: require.Error,
		},
		{
			name: "with empty scope arrays",
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
										ClientId:             "test-client-id",
										DefaultClientScopes:  []string{},
										OptionalClientScopes: []string{},
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

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &keycloakApiAlpha.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			objClient := tt.client(t)

			el := NewPutClientScope(tt.keycloakApiClient(t), objClient)

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
