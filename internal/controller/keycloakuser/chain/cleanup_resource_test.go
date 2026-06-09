package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
)

func TestCleanupResource_Serve(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApiAlpha.AddToScheme(scheme))

	tests := []struct {
		name      string
		user      *keycloakApiAlpha.KeycloakUser
		k8sClient func(t *testing.T) client.Client
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "KeepResource is true - should skip deletion",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:     "testuser",
					KeepResource: true,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - should delete resource",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				user := &keycloakApiAlpha.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-user",
						Namespace: "default",
					},
					Spec: keycloakApiAlpha.KeycloakUserSpec{
						Username:     "testuser",
						KeepResource: false,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - resource not found should not error",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "KeepResource is false - delete fails with error",
			user: &keycloakApiAlpha.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApiAlpha.KeycloakUserSpec{
					Username:     "testuser",
					KeepResource: false,
				},
			},
			k8sClient: func(t *testing.T) client.Client {
				return &fakeClientWithDeleteError{
					Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
				}
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCleanupResource(tt.k8sClient(t))
			err := handler.Serve(
				context.Background(),
				tt.user,
				"",
				&UserContext{},
			)

			tt.wantErr(t, err)
		})
	}
}

// fakeClientWithDeleteError is a fake client that returns an error on Delete.
type fakeClientWithDeleteError struct {
	client.Client
}

func (f *fakeClientWithDeleteError) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return errors.New("delete error")
}
