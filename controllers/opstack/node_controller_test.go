package opstack

import (
	"context"
	"fmt"
	"os"
	"time"

	opstackv1alpha1 "github.com/kotalco/kotal/apis/opstack/v1alpha1"
	sharedAPI "github.com/kotalco/kotal/apis/shared"
	"github.com/kotalco/kotal/controllers/shared"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("OP Stack node controller", func() {
	const (
		sleepTime = 5 * time.Second
		interval  = 2 * time.Second
		timeout   = 2 * time.Minute
	)

	var (
		useExistingCluster = os.Getenv(shared.EnvUseExistingCluster) == "true"
	)

	Context("Joining Base Mainnet with OpRethClient", Ordered, func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "base-mainnet",
			},
		}
		key := types.NamespacedName{
			Name:      "base-node",
			Namespace: ns.Name,
		}

		spec := opstackv1alpha1.NodeSpec{
			Client:           opstackv1alpha1.OpRethClient,
			Network:          "base",
			L1Endpoint:       "http://l1-rpc:8545",
			L1BeaconEndpoint: "http://l1-beacon:5052",
			SyncMode:         opstackv1alpha1.SnapSynchronization,
			Logging:          sharedAPI.InfoLogs,
			RPC:              true,
			WS:               true,
			Engine:           true,
		}

		toCreate := &opstackv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}
		t := true

		nodeOwnerReference := metav1.OwnerReference{
			APIVersion:         "opstack.kotal.io/v1alpha1",
			Kind:               "Node",
			Name:               toCreate.Name,
			Controller:         &t,
			BlockOwnerDeletion: &t,
		}

		It(fmt.Sprintf("should create %s namespace", ns.Name), func() {
			Expect(k8sClient.Create(context.Background(), ns)).Should(Succeed())
		})

		It("should create the OP Stack node", func() {
			if !useExistingCluster {
				toCreate.Default()
			}
			Expect(k8sClient.Create(context.Background(), toCreate)).Should(Succeed())
			time.Sleep(sleepTime)
		})

		It("should get the OP Stack node", func() {
			fetched := &opstackv1alpha1.Node{}
			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Spec.Client).To(Equal(opstackv1alpha1.OpRethClient))
			nodeOwnerReference.UID = fetched.GetUID()
		})

		It("should create shared JWT secret", func() {
			Eventually(func() bool {
				secret := &corev1.Secret{}
				jwtSecretKey := types.NamespacedName{Name: fmt.Sprintf("%s-jwt", key.Name), Namespace: key.Namespace}
				err := k8sClient.Get(context.Background(), jwtSecretKey, secret)
				if err != nil {
					return false
				}
				return len(secret.Data["secret"]) > 0
			}, timeout, interval).Should(BeTrue())
		})

		It("should create node service with P2P, RPC, WS, Engine, and Rollup RPC ports", func() {
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), key, svc)
			}, timeout, interval).Should(Succeed())
			Expect(svc.GetOwnerReferences()).To(ContainElement(nodeOwnerReference))
			
			// We just need to find these ports in the list, we don't need to match the exact slice exactly
			foundP2PUDP := false
			foundP2PTCP := false
			foundRPC := false
			foundWS := false
			foundEngine := false
			foundRollup := false

			for _, port := range svc.Spec.Ports {
				if port.Name == "discovery" && port.Port == int32(opstackv1alpha1.DefaultP2PPort) && port.Protocol == corev1.ProtocolUDP {
					foundP2PUDP = true
				}
				if port.Name == "p2p" && port.Port == int32(opstackv1alpha1.DefaultP2PPort) {
					foundP2PTCP = true
				}
				if port.Name == "rpc" && port.Port == int32(opstackv1alpha1.DefaultRPCPort) {
					foundRPC = true
				}
				if port.Name == "ws" && port.Port == int32(opstackv1alpha1.DefaultWSPort) {
					foundWS = true
				}
				if port.Name == "engine" && port.Port == int32(opstackv1alpha1.DefaultEngineRPCPort) {
					foundEngine = true
				}
				if port.Name == "rollup-rpc" && port.Port == 9545 {
					foundRollup = true
				}
			}

			Expect(foundP2PUDP).To(BeTrue(), "discovery UDP port missing")
			Expect(foundP2PTCP).To(BeTrue(), "p2p TCP port missing")
			Expect(foundRPC).To(BeTrue(), "rpc port missing")
			Expect(foundWS).To(BeTrue(), "ws port missing")
			Expect(foundEngine).To(BeTrue(), "engine port missing")
			Expect(foundRollup).To(BeTrue(), "rollup-rpc port missing")
		})

		It("should create node statefulset with two containers (EL and CL)", func() {
			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), key, sts)
			}, timeout, interval).Should(Succeed())
			Expect(sts.GetOwnerReferences()).To(ContainElement(nodeOwnerReference))
			
			// execution container
			Expect(sts.Spec.Template.Spec.Containers[0].Name).To(Equal("execution"))
			Expect(sts.Spec.Template.Spec.Containers[0].Image).To(Equal(opstackv1alpha1.DefaultOpRethImage))
			
			// op-node sidecar container
			Expect(sts.Spec.Template.Spec.Containers[1].Name).To(Equal("op-node"))
			Expect(sts.Spec.Template.Spec.Containers[1].Image).To(Equal(opstackv1alpha1.DefaultOpNodeImage))
		})

		It("should create node data PVC with correct storage allocation", func() {
			pvc := &corev1.PersistentVolumeClaim{}
			expectedResources := corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(opstackv1alpha1.DefaultMainNetworkFastNodeStorageRequest),
				},
			}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), key, pvc)
			}, timeout, interval).Should(Succeed())
			Expect(pvc.GetOwnerReferences()).To(ContainElement(nodeOwnerReference))
			Expect(pvc.Spec.Resources).To(Equal(expectedResources))
		})

		It("should delete the OP Stack node", func() {
			toDelete := &opstackv1alpha1.Node{}
			Expect(k8sClient.Get(context.Background(), key, toDelete)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), toDelete)).To(Succeed())
			time.Sleep(sleepTime)
		})

		It("should not get the OP Stack node after deletion", func() {
			fetched := &opstackv1alpha1.Node{}
			Expect(k8sClient.Get(context.Background(), key, fetched)).ToNot(Succeed())
		})

		It(fmt.Sprintf("should delete %s namespace", ns.Name), func() {
			Expect(k8sClient.Delete(context.Background(), ns)).Should(Succeed())
		})
	})
})
