package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	v2mocks "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2/mocks"
)

func TestNewRemoveUser(t *testing.T) {
	h := NewRemoveUser(nil)
	require.NotNil(t, h)
}

func TestRemoveUser_ServeRequest(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApiAlpha.KeycloakUser
		mockSetup func(*v2mocks.MockUsersClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - user deleted",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "user not found on FindUserByUsername - skip silently",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakv2.ErrNotFound)
			},
			wantErr: require.NoError,
		},
		{
			name: "user not found on DeleteUser - skip silently",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, keycloakv2.ErrNotFound)
			},
			wantErr: require.NoError,
		},
		{
			name: "preserve resources on deletion annotation set - skip",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
					Annotations: map[string]string{
						"idp.edenlab.io/preserve-resources-on-deletion": "true",
					},
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "FindUserByUsername returns unexpected error",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, errors.New("connection error"))
			},
			wantErr: require.Error,
		},
		{
			name: "DeleteUser returns unexpected error",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, errors.New("delete error"))
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(mockUsers)

			h := NewRemoveUser(&keycloakv2.KeycloakClient{
				Users: mockUsers,
			})

			err := h.ServeRequest(context.Background(), tt.user, "test-realm")
			tt.wantErr(t, err)
		})
	}
}
