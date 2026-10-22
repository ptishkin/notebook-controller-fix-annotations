package tests

import (
	"testing"

	reconcilehelper "github.com/kubeflow/kubeflow/components/common/reconcilehelper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func int32Ptr(i int32) *int32 { return &i }

// TestCopyServiceFields_PreservesForeignAnnotations verifies the
// annotation-merge fix: annotations/labels present only on the live object
// (set by other controllers, e.g. OVN CNI) must survive CopyServiceFields.
// Upstream code did to.Annotations = from.Annotations, wiping them on every
// reconcile and fighting with the controllers that set them.
func TestCopyServiceFields_PreservesForeignAnnotations(t *testing.T) {
	from := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			// desired state has no annotations (as generateService produces)
		},
	}
	to := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				"ovn.example.com/some-annotation": "keep-me",
			},
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	if reconcilehelper.CopyServiceFields(from, to) {
		t.Fatalf("expected no update required when desired state is empty")
	}
	if got := to.Annotations["ovn.example.com/some-annotation"]; got != "keep-me" {
		t.Fatalf("foreign annotation was wiped, got %q", got)
	}
	if got := to.Labels["app"]; got != "test" {
		t.Fatalf("foreign label was wiped, got %q", got)
	}
}

func TestCopyServiceFields_UpdatesOwnedKeys(t *testing.T) {
	from := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"owner": "notebook-controller"},
			Labels:      map[string]string{"statefulset": "nb"},
		},
	}
	to := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"owner":                           "old-value",
				"ovn.example.com/some-annotation": "keep-me",
			},
		},
	}

	if !reconcilehelper.CopyServiceFields(from, to) {
		t.Fatalf("expected update required when owned key differs")
	}
	if got := to.Annotations["owner"]; got != "notebook-controller" {
		t.Fatalf("owned annotation not updated, got %q", got)
	}
	if got := to.Annotations["ovn.example.com/some-annotation"]; got != "keep-me" {
		t.Fatalf("foreign annotation was wiped, got %q", got)
	}
	if got := to.Labels["statefulset"]; got != "nb" {
		t.Fatalf("owned label not merged, got %q", got)
	}
}

func TestCopyStatefulSetFields_PreservesForeignAnnotations(t *testing.T) {
	from := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
		Spec: appsv1.StatefulSetSpec{
			Replicas: int32Ptr(1),
		},
	}
	to := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				"reloader.stakater.com/auto": "true",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: int32Ptr(1),
		},
	}

	if reconcilehelper.CopyStatefulSetFields(from, to) {
		t.Fatalf("expected no update required")
	}
	if got := to.Annotations["reloader.stakater.com/auto"]; got != "true" {
		t.Fatalf("foreign annotation was wiped, got %q", got)
	}
}

func TestCopyStatefulSetFields_UpdatesOwnedKeys(t *testing.T) {
	from := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"kubeflow-resource-stopped": "2026-01-01T00:00:00Z"},
		},
		Spec: appsv1.StatefulSetSpec{Replicas: int32Ptr(0)},
	}
	to := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"foreign": "keep"},
		},
		Spec: appsv1.StatefulSetSpec{Replicas: int32Ptr(1)},
	}

	if !reconcilehelper.CopyStatefulSetFields(from, to) {
		t.Fatalf("expected update required (replicas + annotation differ)")
	}
	if got := to.Annotations["kubeflow-resource-stopped"]; got != "2026-01-01T00:00:00Z" {
		t.Fatalf("owned annotation not updated, got %q", got)
	}
	if got := to.Annotations["foreign"]; got != "keep" {
		t.Fatalf("foreign annotation was wiped, got %q", got)
	}
	if *to.Spec.Replicas != 0 {
		t.Fatalf("replicas not copied, got %d", *to.Spec.Replicas)
	}
}
