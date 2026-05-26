package keycloakv2_test

import (
	"context"
	"testing"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestKeycloakClient_GetServerInfo(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	err = c.Refresh(context.Background())
	require.NoError(t, err)
	info, err := c.GetServerInfo(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)
	require.NotEmpty(t, info.SystemInfo.ServerVersion)
	require.NotEmpty(t, info.ComponentTypes)
	require.NotEmpty(t, info.ProviderTypes)
	require.NotEmpty(t, info.Themes)
}
