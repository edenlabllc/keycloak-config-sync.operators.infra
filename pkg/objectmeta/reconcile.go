package objectmeta

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReconcilePausedAnnotation = "idp.edenlab.io/reconcile-paused"
)

func ReconcilePaused(object metav1.Object) bool {
	if object == nil || isNil(object) {
		return false
	}

	annotations := object.GetAnnotations()
	if annotations == nil {
		return false
	}

	if val, ok := annotations[ReconcilePausedAnnotation]; ok && val == "true" {
		return true
	}

	return false
}

// isNil checks if a value inside the typed-nil pointer interface is not
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return val.IsNil()
	default:
		return false
	}
}
