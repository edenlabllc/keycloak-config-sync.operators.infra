package client

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetCondition is a helper to set a condition on KeycloakClient and update status.
// Each chain step calls this when it succeeds or fails.
func SetCondition(
	ctx context.Context,
	k8sClient client.Client,
	keycloakClient *DataClient,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) error {
	conditionType = fmt.Sprintf("%s.%s",
		helper.RemoveSpecialChar(keycloakClient.Client.Name), conditionType)

	if changed := meta.SetStatusCondition(&keycloakClient.CRD.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: keycloakClient.ReasonScope.Generation,
	}); !changed {
		return nil
	}

	if err := k8sClient.Status().Update(ctx, keycloakClient.CRD); err != nil {
		return fmt.Errorf("failed to update condition %s: %w", conditionType, err)
	}

	return nil
}
