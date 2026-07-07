package role

import (
	"context"
	"errors"
	"testing"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRoleName = "test-role"

func TestRemoveRole_ServeRequest_Success(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil)

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.NoError(t, err)
}

func TestRemoveRole_ServeRequest_NotFound(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, keycloakv2.ErrNotFound)

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.NoError(t, err)
}

func TestRemoveRole_ServeRequest_DeleteError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, errors.New("api error"))

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete realm role")
}
