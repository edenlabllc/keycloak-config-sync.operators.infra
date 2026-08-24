package policy

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetCondition is a helper to set a condition on KeycloakClient and update status.
// The client registration policy is realm scoped, so the condition type is not
// prefixed with a client name.
func SetCondition(
	ctx context.Context,
	k8sClient client.Client,
	policy *DataPolicy,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) error {
	if changed := meta.SetStatusCondition(&policy.CRD.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: policy.Generation,
	}); !changed {
		return nil
	}

	if err := k8sClient.Status().Update(ctx, policy.CRD); err != nil {
		return fmt.Errorf("failed to update condition %s: %w", conditionType, err)
	}

	return nil
}
