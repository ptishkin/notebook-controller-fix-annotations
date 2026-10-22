package reconcile

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deployment reconciles a k8s deployment object.
func Deployment(ctx context.Context, r client.Client, deployment *appsv1.Deployment, log logr.Logger) error {
	foundDeployment := &appsv1.Deployment{}
	justCreated := false
	if err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment); err != nil {
		if apierrs.IsNotFound(err) {
			log.Info("Creating Deployment", "namespace", deployment.Namespace, "name", deployment.Name)
			if err := r.Create(ctx, deployment); err != nil {
				log.Error(err, "unable to create deployment")
				return err
			}
			justCreated = true
		} else {
			log.Error(err, "error getting deployment")
			return err
		}
	}
	if !justCreated && CopyDeploymentSetFields(deployment, foundDeployment) {
		log.Info("Updating Deployment", "namespace", deployment.Namespace, "name", deployment.Name)
		if err := r.Update(ctx, foundDeployment); err != nil {
			log.Error(err, "unable to update deployment")
			return err
		}
	}

	return nil
}

// Service reconciles a k8s service object.
func Service(ctx context.Context, r client.Client, service *corev1.Service, log logr.Logger) error {
	foundService := &corev1.Service{}
	justCreated := false
	if err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService); err != nil {
		if apierrs.IsNotFound(err) {
			log.Info("Creating Service", "namespace", service.Namespace, "name", service.Name)
			if err = r.Create(ctx, service); err != nil {
				log.Error(err, "unable to create service")
				return err
			}
			justCreated = true
		} else {
			log.Error(err, "error getting service")
			return err
		}
	}
	if !justCreated && CopyServiceFields(service, foundService) {
		log.Info("Updating Service\n", "namespace", service.Namespace, "name", service.Name)
		if err := r.Update(ctx, foundService); err != nil {
			log.Error(err, "unable to update Service")
			return err
		}
	}

	return nil
}

// VirtualService reconciles an Istio virtual service object.
func VirtualService(ctx context.Context, r client.Client, virtualServiceName, namespace string, virtualservice *unstructured.Unstructured, log logr.Logger) error {
	foundVirtualService := &unstructured.Unstructured{}
	foundVirtualService.SetAPIVersion("networking.istio.io/v1alpha3")
	foundVirtualService.SetKind("VirtualService")
	justCreated := false
	if err := r.Get(ctx, types.NamespacedName{Name: virtualServiceName, Namespace: namespace}, foundVirtualService); err != nil {
		if apierrs.IsNotFound(err) {
			log.Info("Creating virtual service", "namespace", namespace, "name", virtualServiceName)
			if err := r.Create(ctx, virtualservice); err != nil {
				log.Error(err, "unable to create virtual service")
				return err
			}
			justCreated = true
		} else {
			log.Error(err, "error getting virtual service")
			return err
		}
	}
	if !justCreated && CopyVirtualService(virtualservice, foundVirtualService) {
		log.Info("Updating virtual service", "namespace", namespace, "name", virtualServiceName)
		if err := r.Update(ctx, foundVirtualService); err != nil {
			log.Error(err, "unable to update virtual service")
			return err
		}
	}

	return nil
}

// Reference: https://github.com/pwittrock/kubebuilder-workshop/blob/master/pkg/util/util.go

// mergeStringMap merges keys/values from src into dst without deleting
// keys that exist only in dst (set by other controllers, e.g. CNI sidecars).
// Returns true if dst was changed.
func mergeStringMap(dst *map[string]string, src map[string]string) bool {
	changed := false
	if len(src) == 0 {
		return false
	}
	if *dst == nil {
		*dst = map[string]string{}
	}
	for k, v := range src {
		if (*dst)[k] != v {
			(*dst)[k] = v
			changed = true
		}
	}
	return changed
}

// CopyStatefulSetFields copies the owned fields from one StatefulSet to another
// Returns true if the fields copied from don't match to.
//
// NOTE (annotation-merge fix): labels and annotations are MERGED instead of
// replaced. The upstream implementation assigned from.Labels/from.Annotations
// directly, which wiped any labels/annotations added to the live object by
// other controllers (e.g. CNI, istio, monitoring) on every reconcile, causing
// update-fight loops. With the merge semantics below the controller still
// adds/updates its own keys but never deletes foreign ones.
func CopyStatefulSetFields(from, to *appsv1.StatefulSet) bool {
	requireUpdate := false
	if mergeStringMap(&to.Labels, from.Labels) {
		requireUpdate = true
	}

	if mergeStringMap(&to.Annotations, from.Annotations) {
		requireUpdate = true
	}

	if *from.Spec.Replicas != *to.Spec.Replicas {
		*to.Spec.Replicas = *from.Spec.Replicas
		requireUpdate = true
	}

	if !reflect.DeepEqual(to.Spec.Template.Spec, from.Spec.Template.Spec) {
		requireUpdate = true
	}
	to.Spec.Template.Spec = from.Spec.Template.Spec

	return requireUpdate
}

// CopyDeploymentSetFields follows the same merge semantics as
// CopyStatefulSetFields (see the annotation-merge fix note there).
func CopyDeploymentSetFields(from, to *appsv1.Deployment) bool {
	requireUpdate := false
	if mergeStringMap(&to.Labels, from.Labels) {
		requireUpdate = true
	}

	if mergeStringMap(&to.Annotations, from.Annotations) {
		requireUpdate = true
	}

	if from.Spec.Replicas != to.Spec.Replicas {
		to.Spec.Replicas = from.Spec.Replicas
		requireUpdate = true
	}

	if !reflect.DeepEqual(to.Spec.Template.Spec, from.Spec.Template.Spec) {
		requireUpdate = true
	}
	to.Spec.Template.Spec = from.Spec.Template.Spec

	return requireUpdate
}

// CopyServiceFields copies the owned fields from one Service to another.
//
// NOTE (annotation-merge fix): labels and annotations are MERGED instead of
// replaced (same semantics as CopyStatefulSetFields). The upstream code did
// to.Annotations = from.Annotations, and since generateService() creates the
// desired Service without any annotations, every reconcile wiped annotations
// added by other controllers (e.g. OVN CNI) and pushed them back via Update,
// creating an endless patch/update fight visible in the audit log.
func CopyServiceFields(from, to *corev1.Service) bool {
	requireUpdate := false
	if mergeStringMap(&to.Labels, from.Labels) {
		requireUpdate = true
	}

	if mergeStringMap(&to.Annotations, from.Annotations) {
		requireUpdate = true
	}

	// Don't copy the entire Spec, because we can't overwrite the clusterIp field

	if !reflect.DeepEqual(to.Spec.Selector, from.Spec.Selector) {
		requireUpdate = true
	}
	to.Spec.Selector = from.Spec.Selector

	if !reflect.DeepEqual(to.Spec.Ports, from.Spec.Ports) {
		requireUpdate = true
	}
	to.Spec.Ports = from.Spec.Ports

	return requireUpdate
}

// Copy configuration related fields to another instance and returns true if there
// is a diff and thus needs to update.
func CopyVirtualService(from, to *unstructured.Unstructured) bool {
	fromSpec, found, err := unstructured.NestedMap(from.Object, "spec")
	if !found {
		return false
	}
	if err != nil {
		return false
	}

	toSpec, found, err := unstructured.NestedMap(to.Object, "spec")
	if !found || err != nil {
		unstructured.SetNestedMap(to.Object, fromSpec, "spec")
		return true
	}

	requiresUpdate := !reflect.DeepEqual(fromSpec, toSpec)
	if requiresUpdate {
		unstructured.SetNestedMap(to.Object, fromSpec, "spec")
	}
	return requiresUpdate
}
