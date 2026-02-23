package v1alpha1

import "github.com/kotalco/kotal/apis/shared"

var (
	// DefaultAPIs is the default rpc, ws APIs
	DefaultAPIs []API = []API{Web3API, ETHAPI, NetworkAPI, RollupAPI}
	// DefaultOrigins is the default origins
	DefaultOrigins []string = []string{"*"}
	// DefaltReplicas is the default replicas
	DefaltReplicas uint = 1
)

const (
	// DefaultOpGethImage is op-geth image
	DefaultOpGethImage = "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-geth:v1.101608.0"
	// DefaultOpRethImage is the op-reth image
	DefaultOpRethImage = "ghcr.io/paradigmxyz/op-reth:v1.10.2"
	// DefaultBaseRethNodeImage is the base-reth-node image
	DefaultBaseRethNodeImage = "ghcr.io/base/node-reth:v0.4.1"
	// DefaultOpNodeImage is the op-node rollup driver image
	DefaultOpNodeImage = "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:v1.16.6"
)

// Node defaults
const (
	// DefaultLogging is the default logging verbosity level
	DefaultLogging = shared.InfoLogs
	// DefaultP2PPort is the default p2p port
	DefaultP2PPort uint = 30303
	// DefaultEngineRPCPort is the default engine rpc port
	DefaultEngineRPCPort uint = 8551
	// DefaultRPCPort is the default rpc port
	DefaultRPCPort uint = 8545
	// DefaultWSPort is the default ws port
	DefaultWSPort uint = 8546
)

// Resources
const (
	// DefaultPublicNetworkNodeCPURequest is the cpu requested by public network node
	DefaultPublicNetworkNodeCPURequest = "4"
	// DefaultPublicNetworkNodeCPULimit is the cpu limit for public network node
	DefaultPublicNetworkNodeCPULimit = "8"
	// DefaultPublicNetworkNodeMemoryRequest is the Memory requested by public network node
	DefaultPublicNetworkNodeMemoryRequest = "16Gi"
	// DefaultPublicNetworkNodeMemoryLimit is the Memory limit for public network node
	DefaultPublicNetworkNodeMemoryLimit = "32Gi"
	
	// DefaultMainNetworkFullNodeStorageRequest is the Storage requested by main network archive node
	DefaultMainNetworkFullNodeStorageRequest = "6Ti"
	// DefaultMainNetworkFastNodeStorageRequest is the Storage requested by main network full node
	DefaultMainNetworkFastNodeStorageRequest = "750Gi"
	// DefaultTestNetworkStorageRequest is the Storage requested by main network full node
	DefaultTestNetworkStorageRequest = "50Gi"
)