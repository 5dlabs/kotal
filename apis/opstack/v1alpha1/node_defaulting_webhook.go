package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-opstack-kotal-io-v1alpha1-node,mutating=true,failurePolicy=fail,groups=opstack.kotal.io,resources=nodes,verbs=create;update,versions=v1alpha1,name=mutate-opstack-v1alpha1-node.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.Defaulter = &Node{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (n *Node) Default() {
	if n.Spec.Image == "" {
		var image string

		switch n.Spec.Client {
		case OpGethClient:
			image = DefaultOpGethImage
		case OpRethClient:
			image = DefaultOpRethImage
		case BaseRethNodeClient:
			image = DefaultBaseRethNodeImage
		}

		n.Spec.Image = image
	}

	if n.Spec.NodeImage == "" {
		n.Spec.NodeImage = DefaultOpNodeImage
	}

	if n.Spec.Replicas == nil {
		replicas := DefaltReplicas
		n.Spec.Replicas = &replicas
	}

	if n.Spec.P2PPort == 0 {
		n.Spec.P2PPort = DefaultP2PPort
	}

	if n.Spec.SyncMode == "" {
		if n.Spec.Client == OpGethClient {
			n.Spec.SyncMode = SnapSynchronization
		} else {
			n.Spec.SyncMode = FastSynchronization
		}
	}

	// must be called after defaulting sync mode because it's depending on its value
	n.DefaultNodeResources()

	// op-reth doesn't support host whitelisting
	if len(n.Spec.Hosts) == 0 && n.Spec.Client != OpRethClient && n.Spec.Client != BaseRethNodeClient {
		n.Spec.Hosts = DefaultOrigins
	}

	// op-reth doesn't support cors domains
	if len(n.Spec.CORSDomains) == 0 && n.Spec.Client != OpRethClient && n.Spec.Client != BaseRethNodeClient {
		n.Spec.CORSDomains = DefaultOrigins
	}

	if n.Spec.EnginePort == 0 {
		n.Spec.EnginePort = DefaultEngineRPCPort
	}

	// Engine API is always required for OP Stack since op-node connects via it
	n.Spec.Engine = true

	if n.Spec.RPCPort == 0 {
		n.Spec.RPCPort = DefaultRPCPort
	}

	if len(n.Spec.RPCAPI) == 0 {
		n.Spec.RPCAPI = DefaultAPIs
	}

	if n.Spec.WSPort == 0 {
		n.Spec.WSPort = DefaultWSPort
	}

	if len(n.Spec.WSAPI) == 0 {
		n.Spec.WSAPI = DefaultAPIs
	}

	if n.Spec.Logging == "" {
		n.Spec.Logging = DefaultLogging
	}

}

// DefaultNodeResources defaults node cpu, memory and storage resources
func (n *Node) DefaultNodeResources() {
	var cpu, cpuLimit, memory, memoryLimit, storage string

	if n.Spec.Resources.CPU == "" {
		cpu = DefaultPublicNetworkNodeCPURequest
		n.Spec.Resources.CPU = cpu
	}

	if n.Spec.Resources.CPULimit == "" {
		cpuLimit = DefaultPublicNetworkNodeCPULimit
		n.Spec.Resources.CPULimit = cpuLimit
	}

	if n.Spec.Resources.Memory == "" {
		memory = DefaultPublicNetworkNodeMemoryRequest
		n.Spec.Resources.Memory = memory
	}

	if n.Spec.Resources.MemoryLimit == "" {
		memoryLimit = DefaultPublicNetworkNodeMemoryLimit
		n.Spec.Resources.MemoryLimit = memoryLimit
	}

	if n.Spec.Resources.Storage == "" {
		if n.Spec.SyncMode == FastSynchronization || n.Spec.SyncMode == SnapSynchronization {
			storage = DefaultMainNetworkFastNodeStorageRequest
		} else if n.Spec.SyncMode == FullSynchronization {
			storage = DefaultMainNetworkFullNodeStorageRequest
		} else {
			storage = DefaultTestNetworkStorageRequest
		}
		n.Spec.Resources.Storage = storage
	}
}