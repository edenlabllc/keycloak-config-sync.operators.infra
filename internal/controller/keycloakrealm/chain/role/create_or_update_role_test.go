package role

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2/mocks"
)

func TestCreateOrUpdateRole_Serve_CreateNew(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName
	role.Description = "Test description"
	role.Composite = false
	role.Attributes = map[string][]string{"key": {"val"}}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, keycloakv2.ErrNotFound).Once()

	desc := "Test description"
	composite := false
	attrs := map[string][]string{"key": {"val"}}

	mockRoles.EXPECT().CreateRealmRole(
		context.Background(), "test-realm",
		keycloakv2.RoleRepresentation{
			Name:        ptr.To(testRoleName),
			Description: &desc,
			Composite:   &composite,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}, nil, nil).Once()

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_UpdateExisting(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName
	role.Description = "Updated description"
	role.Composite = true
	role.Attributes = map[string][]string{"key": {"new-val"}}

	existingRole := &keycloakv2.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(existingRole, nil, nil)

	desc := "Updated description"
	composite := true
	attrs := map[string][]string{"key": {"new-val"}}

	mockRoles.EXPECT().UpdateRealmRole(
		context.Background(), "test-realm", testRoleName,
		keycloakv2.RoleRepresentation{
			Id:          ptr.To("role-id-123"),
			Name:        ptr.To(testRoleName),
			Description: &desc,
			Composite:   &composite,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_GetRealmRoleError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get realm role")
}

func TestCreateOrUpdateRole_Serve_CreateError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, keycloakv2.ErrNotFound)

	var nilAttrs map[string][]string

	mockRoles.EXPECT().CreateRealmRole(
		context.Background(), "test-realm",
		keycloakv2.RoleRepresentation{
			Name:        ptr.To(testRoleName),
			Description: ptr.To(""),
			Composite:   ptr.To(false),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create realm role")
}

func TestCreateOrUpdateRole_Serve_UpdateError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakv2.KeycloakClient{Roles: mockRoles}

	role := &keycloakApiAlpha.Role{}
	role.Name = testRoleName

	existingRole := &keycloakv2.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(existingRole, nil, nil)

	var nilAttrs map[string][]string

	mockRoles.EXPECT().UpdateRealmRole(
		context.Background(), "test-realm", testRoleName,
		keycloakv2.RoleRepresentation{
			Id:          ptr.To("role-id-123"),
			Name:        ptr.To(testRoleName),
			Description: ptr.To(""),
			Composite:   ptr.To(false),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update realm role")
}
