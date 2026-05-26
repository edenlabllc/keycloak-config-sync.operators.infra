package objectmeta

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	PreserveResourcesOnDeletionAnnotation = "edp.edenlab.io/preserve-resources-on-deletion"
	PreserveResourcesNoDeleteAnnotation   = "edp.edenlab.io/preserve-resources-no-delete"
)

// PreserveResourcesOnDeletion returns true if the object has annotation
// that indicates that resources must not be deleted.
func PreserveResourcesOnDeletion(object metav1.Object) bool {
	return object.GetAnnotations()[PreserveResourcesOnDeletionAnnotation] == "true" ||
		object.GetAnnotations()[PreserveResourcesNoDeleteAnnotation] == "true"
}
