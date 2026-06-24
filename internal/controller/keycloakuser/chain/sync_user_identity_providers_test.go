package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	v2mocks "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2/mocks"
)

func TestNewSyncUserIdentityProviders(t *testing.T) {
	h := NewSyncUserIdentityProviders(nil)
	require.NotNil(t, h)
}

func TestSyncUserIdentityProviders_Serve(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApiAlpha.KeycloakUser
		userCtx   *UserContext
		mockSetup func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - nil identity providers skips sync",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			userCtx:   &UserContext{UserID: "user-1"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "success - add missing identity provider",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"google"},
				},
			},
			userCtx: &UserContext{UserID: "user-2"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-2").
					Return(nil, nil, nil)
				idp.EXPECT().GetIdentityProvider(context.Background(), "test-realm", "google").
					Return(&keycloakv2.IdentityProviderRepresentation{}, &keycloakv2.Response{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil)
				u.EXPECT().CreateUserFederatedIdentity(
					context.Background(),
					"test-realm",
					"user-2",
					"google",
					keycloakv2.FederatedIdentityRepresentation{
						IdentityProvider: ptr.To("google"),
						UserId:           ptr.To("user-2"),
						UserName:         ptr.To("testuser"),
					},
				).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - remove extra identity provider",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{},
				},
			},
			userCtx: &UserContext{UserID: "user-3"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-3").
					Return([]keycloakv2.FederatedIdentityRepresentation{
						{IdentityProvider: ptr.To("github")},
					}, nil, nil)
				u.EXPECT().DeleteUserFederatedIdentity(context.Background(), "test-realm", "user-3", "github").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - existing provider already linked, no action",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"google"},
				},
			},
			userCtx: &UserContext{UserID: "user-4"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-4").
					Return([]keycloakv2.FederatedIdentityRepresentation{
						{IdentityProvider: ptr.To("google")},
					}, nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "error - GetUserFederatedIdentities fails",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"google"},
				},
			},
			userCtx: &UserContext{UserID: "user-err"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-err").
					Return(nil, nil, errors.New("api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get user federated identities")
			},
		},
		{
			name: "error - identity provider does not exist (404)",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"nonexistent"},
				},
			},
			userCtx: &UserContext{UserID: "user-err2"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-err2").
					Return(nil, nil, nil)
				idp.EXPECT().GetIdentityProvider(context.Background(), "test-realm", "nonexistent").
					Return(nil, &keycloakv2.Response{
						HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
					}, &keycloakv2.ApiError{Code: 404})
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "identity provider")
				assert.Contains(t, err.Error(), "does not exist")
			},
		},
		{
			name: "error - GetIdentityProvider fails with non-404 error",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"broken-idp"},
				},
			},
			userCtx: &UserContext{UserID: "user-err3"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-err3").
					Return(nil, nil, nil)
				idp.EXPECT().GetIdentityProvider(context.Background(), "test-realm", "broken-idp").
					Return(nil, nil, errors.New("connection refused"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to check if identity provider")
			},
		},
		{
			name: "error - CreateUserFederatedIdentity fails",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"google"},
				},
			},
			userCtx: &UserContext{UserID: "user-err4"},
			mockSetup: func(u *v2mocks.MockUsersClient, idp *v2mocks.MockIdentityProvidersClient) {
				u.EXPECT().GetUserFederatedIdentities(context.Background(), "test-realm", "user-err4").
					Return(nil, nil, nil)
				idp.EXPECT().GetIdentityProvider(context.Background(), "test-realm", "google").
					Return(&keycloakv2.IdentityProviderRepresentation{}, &keycloakv2.Response{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil)
				u.EXPECT().CreateUserFederatedIdentity(
					context.Background(),
					"test-realm",
					"user-err4",
					"google",
					keycloakv2.FederatedIdentityRepresentation{
						IdentityProvider: ptr.To("google"),
						UserId:           ptr.To("user-err4"),
						UserName:         ptr.To("testuser"),
					},
				).Return(nil, errors.New("create error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to add user to identity provider")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			mockIDP := v2mocks.NewMockIdentityProvidersClient(t)
			tt.mockSetup(mockUsers, mockIDP)

			h := NewSyncUserIdentityProviders(&keycloakv2.KeycloakClient{
				Users:             mockUsers,
				IdentityProviders: mockIDP,
			})

			err := h.Serve(
				context.Background(),
				tt.user,
				"test-realm",
				tt.userCtx,
			)

			tt.wantErr(t, err)
		})
	}
}
