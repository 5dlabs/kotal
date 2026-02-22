package ethereum

import (
	"fmt"

	ethereumv1alpha1 "github.com/kotalco/kotal/apis/ethereum/v1alpha1"
	sharedAPI "github.com/kotalco/kotal/apis/shared"
	"github.com/kotalco/kotal/controllers/shared"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Reth Client", func() {

	enode := ethereumv1alpha1.Enode("enode://2281549869465d98e90cebc45e1d6834a01465a990add7bcf07a49287e7e66b50ca27f9c70a46190cef7ad746dd5d5b6b9dfee0c9954104c8e9bd0d42758ec58@10.5.0.2:30300")

	Context("general", func() {
		node := &ethereumv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "reth-general",
			},
			Spec: ethereumv1alpha1.NodeSpec{
				Client:  ethereumv1alpha1.RethClient,
				Network: ethereumv1alpha1.MainNetwork,
			},
		}
		client, _ := NewClient(node)

		It("should return the correct home directory", func() {
			Expect(client.HomeDir()).To(Equal(RethHomeDir))
		})

		It("should return empty string for EncodeStaticNodes", func() {
			Expect(client.EncodeStaticNodes()).To(Equal(""))
		})

		It("should return error from Genesis()", func() {
			_, err := client.Genesis()
			Expect(err).NotTo(BeNil())
		})
	})

	Context("joining mainnet with all options", func() {
		node := &ethereumv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "reth-mainnet",
			},
			Spec: ethereumv1alpha1.NodeSpec{
				Network:                  ethereumv1alpha1.MainNetwork,
				Client:                   ethereumv1alpha1.RethClient,
				Bootnodes:                []ethereumv1alpha1.Enode{enode},
				NodePrivateKeySecretName: "reth-nodekey",
				P2PPort:                  30303,
				SyncMode:                 ethereumv1alpha1.SnapSynchronization,
				Logging:                  sharedAPI.WarnLogs,
				RPC:                      true,
				RPCPort:                  8545,
				RPCAPI: []ethereumv1alpha1.API{
					ethereumv1alpha1.ETHAPI,
					ethereumv1alpha1.NetworkAPI,
					ethereumv1alpha1.Web3API,
				},
				Engine:        true,
				EnginePort:    8551,
				JWTSecretName: "jwt-secret",
				WS:            true,
				WSPort:        8546,
				WSAPI: []ethereumv1alpha1.API{
					ethereumv1alpha1.ETHAPI,
					ethereumv1alpha1.NetworkAPI,
				},
			},
		}

		It("should generate correct arguments", func() {
			client, err := NewClient(node)
			Expect(err).To(BeNil())
			Expect(client.Args()).To(ContainElements(
				"node",
				"--datadir",
				shared.PathData(client.HomeDir()),
				"--port",
				"30303",
				"--chain",
				ethereumv1alpha1.MainNetwork,
				"--log.stdout.filter",
				"warn",
				"--p2p-secret-key",
				fmt.Sprintf("%s/nodekey", shared.PathSecrets(client.HomeDir())),
				"--bootnodes",
				string(enode),
				"--http",
				"--http.addr",
				"0.0.0.0",
				"--http.port",
				"8545",
				"--http.api",
				"eth,net,web3",
				"--ws",
				"--ws.addr",
				"0.0.0.0",
				"--ws.port",
				"8546",
				"--ws.api",
				"eth,net",
				"--authrpc.addr",
				"0.0.0.0",
				"--authrpc.port",
				"8551",
				"--authrpc.jwtsecret",
				fmt.Sprintf("%s/jwt.secret", shared.PathSecrets(client.HomeDir())),
			))
		})

		It("should not include --full flag for snap sync", func() {
			client, err := NewClient(node)
			Expect(err).To(BeNil())
			Expect(client.Args()).NotTo(ContainElement("--full"))
		})
	})

	Context("full sync mode", func() {
		node := &ethereumv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "reth-full",
			},
			Spec: ethereumv1alpha1.NodeSpec{
				Network:  ethereumv1alpha1.MainNetwork,
				Client:   ethereumv1alpha1.RethClient,
				P2PPort:  30303,
				SyncMode: ethereumv1alpha1.FullSynchronization,
			},
		}

		It("should include --full flag for full sync", func() {
			client, err := NewClient(node)
			Expect(err).To(BeNil())
			Expect(client.Args()).To(ContainElement("--full"))
		})
	})

	Context("logging levels", func() {
		logLevels := []struct {
			level    sharedAPI.VerbosityLevel
			expected string
		}{
			{sharedAPI.NoLogs, "off"},
			{sharedAPI.ErrorLogs, "error"},
			{sharedAPI.WarnLogs, "warn"},
			{sharedAPI.InfoLogs, "info"},
			{sharedAPI.DebugLogs, "debug"},
			{sharedAPI.TraceLogs, "trace"},
		}

		for _, ll := range logLevels {
			ll := ll
			It(fmt.Sprintf("should map %s to %s", ll.level, ll.expected), func() {
				node := &ethereumv1alpha1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "reth-log"},
					Spec: ethereumv1alpha1.NodeSpec{
						Network: ethereumv1alpha1.MainNetwork,
						Client:  ethereumv1alpha1.RethClient,
						P2PPort: 30303,
						Logging: ll.level,
					},
				}
				client, err := NewClient(node)
				Expect(err).To(BeNil())
				Expect(client.Args()).To(ContainElements("--log.stdout.filter", ll.expected))
			})
		}
	})

	Context("engine disabled", func() {
		node := &ethereumv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "reth-no-engine",
			},
			Spec: ethereumv1alpha1.NodeSpec{
				Network:    ethereumv1alpha1.MainNetwork,
				Client:     ethereumv1alpha1.RethClient,
				P2PPort:    30303,
				Engine:     false,
				EnginePort: 8551,
			},
		}

		It("should not include --authrpc.jwtsecret when engine is disabled", func() {
			client, err := NewClient(node)
			Expect(err).To(BeNil())
			Expect(client.Args()).NotTo(ContainElement("--authrpc.jwtsecret"))
		})

		It("should still include authrpc addr/port with localhost binding", func() {
			client, err := NewClient(node)
			Expect(err).To(BeNil())
			Expect(client.Args()).To(ContainElements(
				"--authrpc.addr",
				"127.0.0.1",
				"--authrpc.port",
				"8551",
			))
		})
	})
})
