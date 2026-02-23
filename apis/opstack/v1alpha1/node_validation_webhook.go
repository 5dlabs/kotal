package v1alpha1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-opstack-kotal-io-v1alpha1-node,mutating=false,failurePolicy=fail,groups=opstack.kotal.io,resources=nodes,versions=v1alpha1,name=validate-opstack-v1alpha1-node.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.Validator = &Node{}

// validate validates a node with a given path
func (n *Node) validate() field.ErrorList {
	var nodeErrors field.ErrorList

	path := field.NewPath("spec")

	if n.Spec.L1Endpoint == "" {
		err := field.Invalid(path.Child("l1Endpoint"), "", "must provide l1Endpoint for OP Stack nodes")
		nodeErrors = append(nodeErrors, err)
	}

	if n.Spec.L1BeaconEndpoint == "" {
		err := field.Invalid(path.Child("l1BeaconEndpoint"), "", "must provide l1BeaconEndpoint for OP Stack nodes")
		nodeErrors = append(nodeErrors, err)
	}

	// validate rpc must be enabled if ws is enabled (for some clients)
	// op-reth doesn't support hosts whitelisting
	if len(n.Spec.Hosts) > 0 && (n.Spec.Client == OpRethClient || n.Spec.Client == BaseRethNodeClient) {
		err := field.Invalid(path.Child("client"), n.Spec.Client, "client doesn't support hosts whitelisting")
		nodeErrors = append(nodeErrors, err)
	}

	// validate op-reth doesn't support CORS domains
	if len(n.Spec.CORSDomains) > 0 && (n.Spec.Client == OpRethClient || n.Spec.Client == BaseRethNodeClient) {
		err := field.Invalid(path.Child("client"), n.Spec.Client, "client doesn't support CORS domains")
		nodeErrors = append(nodeErrors, err)
	}

	return nodeErrors
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (n *Node) ValidateCreate() (admission.Warnings, error) {
	nodeErrors := n.validate()
	if len(nodeErrors) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "opstack.kotal.io", Kind: "Node"},
			n.Name,
			nodeErrors,
		)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (n *Node) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nodeErrors := n.validate()
	if len(nodeErrors) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "opstack.kotal.io", Kind: "Node"},
			n.Name,
			nodeErrors,
		)
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (n *Node) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}