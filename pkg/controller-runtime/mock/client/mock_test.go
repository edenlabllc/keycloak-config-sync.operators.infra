package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMockClient(t *testing.T) {
	t.Parallel()

	mockClient := &Client{}

	assert.Implements(t, (*client.Client)(nil), mockClient)
}
