package v1alpha1

import (
	"github.com/kotalco/kotal/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeStatus defines the observed state of Node
type NodeStatus struct {
	// Consensus is network consensus algorithm
	Consensus string `json:"consensus,omitempty"`
	// Network is the network this node is joining
	Network string `json:"network,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Node is the Schema for the nodes API
// +kubebuilder:printcolumn:name="Network",type=string,JSONPath=".spec.network"
// +kubebuilder:printcolumn:name="Client",type=string,JSONPath=".spec.client"
// +kubebuilder:printcolumn:name="L1Endpoint",type=string,JSONPath=".spec.l1Endpoint"
type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeList contains a list of Node
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Node `json:"items"`
}

// NodeSpec is the specification of the node
type NodeSpec struct {
	// Network specifies the network to join
	// +kubebuilder:validation:Enum=base;base-sepolia;op-mainnet;op-sepolia
	Network string `json:"network"`

	// Client is the execution client running on the node
	// +kubebuilder:validation:Enum=op-geth;op-reth;base-reth-node
	Client ExecutionClient `json:"client"`

	// Image is Ethereum node client image
	Image string `json:"image,omitempty"`

	// ExtraArgs is extra arguments to pass down to the execution client cli
	ExtraArgs shared.ExtraArgs `json:"extraArgs,omitempty"`

	// NodeImage is the rollup driver (op-node) image
	NodeImage string `json:"nodeImage,omitempty"`

	// NodeExtraArgs is extra arguments to pass down to the rollup driver (op-node) cli
	NodeExtraArgs shared.ExtraArgs `json:"nodeExtraArgs,omitempty"`

	// Replicas is number of replicas
	// +kubebuilder:validation:Enum=0;1
	Replicas *uint `json:"replicas,omitempty"`

	// L1Endpoint is the RPC endpoint of the L1 Ethereum node
	L1Endpoint string `json:"l1Endpoint"`

	// L1BeaconEndpoint is the REST endpoint of the L1 Beacon node
	L1BeaconEndpoint string `json:"l1BeaconEndpoint"`

	// P2PPort is port used for peer to peer communication
	P2PPort uint `json:"p2pPort,omitempty"`

	// SyncMode is the node synchronization mode
	SyncMode SynchronizationMode `json:"syncMode,omitempty"`

	// Logging is logging verbosity level
	// +kubebuilder:validation:Enum=off;fatal;error;warn;info;debug;trace;all
	Logging shared.VerbosityLevel `json:"logging,omitempty"`

	// Hosts is a list of hostnames to whitelist for RPC access
	// +listType=set
	Hosts []string `json:"hosts,omitempty"`

	// CORSDomains is the domains from which to accept cross origin requests
	// +listType=set
	CORSDomains []string `json:"corsDomains,omitempty"`

	// Engine enables authenticated Engine RPC APIs
	Engine bool `json:"engine,omitempty"`

	// EnginePort is engine authenticated RPC APIs port
	EnginePort uint `json:"enginePort,omitempty"`

	// JWTSecretName is kubernetes secret name holding JWT secret
	JWTSecretName string `json:"jwtSecretName,omitempty"`

	// RPC is whether HTTP-RPC server is enabled or not
	RPC bool `json:"rpc,omitempty"`

	// RPCPort is HTTP-RPC server listening port
	RPCPort uint `json:"rpcPort,omitempty"`

	// RPCAPI is a list of rpc services to enable
	// +listType=set
	RPCAPI []API `json:"rpcAPI,omitempty"`

	// WS is whether web socket server is enabled or not
	WS bool `json:"ws,omitempty"`

	// WSPort is the web socket server listening port
	WSPort uint `json:"wsPort,omitempty"`

	// WSAPI is a list of WS services to enable
	// +listType=set
	WSAPI []API `json:"wsAPI,omitempty"`

	// Resources is node compute and storage resources
	shared.Resources `json:"resources,omitempty"`
}

// SynchronizationMode is the node synchronization mode
// +kubebuilder:validation:Enum=fast;full;snap
type SynchronizationMode string

const (
	// SnapSynchronization is the snap synchronization mode
	SnapSynchronization SynchronizationMode = "snap"

	// FastSynchronization is the fast synchronization mode
	FastSynchronization SynchronizationMode = "fast"

	// FullSynchronization is full archival synchronization mode
	FullSynchronization SynchronizationMode = "full"
)

	// API is RPC API to be exposed by RPC or web socket server
// +kubebuilder:validation:Enum=admin;clique;debug;eea;eth;ibft;miner;net;perm;plugins;priv;txpool;web3;rollup
type API string

const (
	// ETHAPI is ethereum API
	ETHAPI API = "eth"

	// NetworkAPI is network API
	NetworkAPI API = "net"

	// Web3API is web3 API
	Web3API API = "web3"
	
	// RollupAPI is the rollup API
	RollupAPI API = "rollup"
)

// ExecutionClient is the execution client running on a given node
type ExecutionClient string

const (
	// OpGethClient is the op-geth client
	OpGethClient ExecutionClient = "op-geth"
	// OpRethClient is the op-reth client
	OpRethClient ExecutionClient = "op-reth"
	// BaseRethNodeClient is the base-reth-node client (supports Flashblocks)
	BaseRethNodeClient ExecutionClient = "base-reth-node"
)

func init() {
	SchemeBuilder.Register(&Node{}, &NodeList{})
}