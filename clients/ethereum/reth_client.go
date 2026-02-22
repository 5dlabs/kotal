package ethereum

import (
	"fmt"
	"strings"

	ethereumv1alpha1 "github.com/kotalco/kotal/apis/ethereum/v1alpha1"
	sharedAPI "github.com/kotalco/kotal/apis/shared"
	"github.com/kotalco/kotal/controllers/shared"
	corev1 "k8s.io/api/core/v1"
)

// RethClient is the Rust Ethereum client
// https://github.com/paradigmxyz/reth
type RethClient struct {
	node *ethereumv1alpha1.Node
}

const (
	// RethHomeDir is the Reth docker image home directory
	RethHomeDir = "/root"
)

var (
	rethVerbosityLevels = map[sharedAPI.VerbosityLevel]string{
		sharedAPI.NoLogs:    "off",
		sharedAPI.ErrorLogs: "error",
		sharedAPI.WarnLogs:  "warn",
		sharedAPI.InfoLogs:  "info",
		sharedAPI.DebugLogs: "debug",
		sharedAPI.TraceLogs: "trace",
	}
)

// HomeDir returns the Reth docker image home directory
func (r *RethClient) HomeDir() string {
	return RethHomeDir
}

// Command returns nil — Reth uses the default image entrypoint
func (r *RethClient) Command() []string {
	return nil
}

// Env returns nil — no special environment variables required
func (r *RethClient) Env() []corev1.EnvVar {
	return nil
}

// Args returns the command-line arguments for the Reth node
func (r *RethClient) Args() (args []string) {
	node := r.node

	args = append(args, "node")
	args = append(args, "--datadir", shared.PathData(r.HomeDir()))
	args = append(args, "--port", fmt.Sprintf("%d", node.Spec.P2PPort))

	if node.Spec.Network != "" {
		args = append(args, "--chain", node.Spec.Network)
	}

	if node.Spec.SyncMode == ethereumv1alpha1.FullSynchronization {
		args = append(args, "--full")
	}

	if node.Spec.Logging != "" {
		if level, ok := rethVerbosityLevels[node.Spec.Logging]; ok {
			args = append(args, "--log.stdout.filter", level)
		}
	}

	if node.Spec.NodePrivateKeySecretName != "" {
		args = append(args, "--p2p-secret-key", fmt.Sprintf("%s/nodekey", shared.PathSecrets(r.HomeDir())))
	}

	if len(node.Spec.Bootnodes) != 0 {
		bootnodes := []string{}
		for _, bootnode := range node.Spec.Bootnodes {
			bootnodes = append(bootnodes, string(bootnode))
		}
		args = append(args, "--bootnodes", strings.Join(bootnodes, ","))
	}

	if node.Spec.RPC {
		args = append(args, "--http")
		args = append(args, "--http.addr", shared.Host(node.Spec.RPC))
		args = append(args, "--http.port", fmt.Sprintf("%d", node.Spec.RPCPort))
		if len(node.Spec.RPCAPI) != 0 {
			apis := []string{}
			for _, api := range node.Spec.RPCAPI {
				apis = append(apis, string(api))
			}
			args = append(args, "--http.api", strings.Join(apis, ","))
		}
	}

	if node.Spec.WS {
		args = append(args, "--ws")
		args = append(args, "--ws.addr", shared.Host(node.Spec.WS))
		args = append(args, "--ws.port", fmt.Sprintf("%d", node.Spec.WSPort))
		if len(node.Spec.WSAPI) != 0 {
			apis := []string{}
			for _, api := range node.Spec.WSAPI {
				apis = append(apis, string(api))
			}
			args = append(args, "--ws.api", strings.Join(apis, ","))
		}
	}

	args = append(args, "--authrpc.addr", shared.Host(node.Spec.Engine))
	args = append(args, "--authrpc.port", fmt.Sprintf("%d", node.Spec.EnginePort))

	if node.Spec.Engine {
		jwtSecretPath := fmt.Sprintf("%s/jwt.secret", shared.PathSecrets(r.HomeDir()))
		args = append(args, "--authrpc.jwtsecret", jwtSecretPath)
	}

	return args
}

// Genesis returns an error because Reth does not support private network genesis
func (r *RethClient) Genesis() (string, error) {
	return "", fmt.Errorf("reth does not support private network genesis")
}

// EncodeStaticNodes returns an empty string — Reth uses --trusted-peers (not yet wired up)
func (r *RethClient) EncodeStaticNodes() string {
	return ""
}
