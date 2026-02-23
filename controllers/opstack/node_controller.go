package opstack

import (
	"context"
	"fmt"

	"math/rand"
	
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	opstackv1alpha1 "github.com/kotalco/kotal/apis/opstack/v1alpha1"
	"github.com/kotalco/kotal/controllers/shared"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opstack.kotal.io,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opstack.kotal.io,resources=nodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=watch;get;list;create;update;delete
// +kubebuilder:rbac:groups=core,resources=secrets;services;configmaps;persistentvolumeclaims,verbs=watch;get;create;update;list;delete

// Reconcile reconciles opstack networks
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	defer shared.IgnoreConflicts(&err)

	var node opstackv1alpha1.Node

	if err = r.Client.Get(ctx, req.NamespacedName, &node); err != nil {
		err = client.IgnoreNotFound(err)
		return
	}

	// default the node if webhooks are disabled
	if !shared.IsWebhookEnabled() {
		node.Default()
	}

	shared.UpdateLabels(&node, string(node.Spec.Client), node.Spec.Network)

	if err = r.reconcilePVC(ctx, &node); err != nil {
		return
	}

	_, err = r.reconcileService(ctx, &node)
	if err != nil {
		return
	}

	// create shared JWT secret between EL and op-node
	var jwtSecretName string
	if jwtSecretName, err = r.reconcileSecret(ctx, &node); err != nil {
		return
	}
	
	// ensure the node uses this generated secret
	node.Spec.JWTSecretName = jwtSecretName

	if err = r.reconcileStatefulSet(ctx, &node); err != nil {
		return
	}

	if err = r.updateStatus(ctx, &node, ""); err != nil {
		return
	}

	return ctrl.Result{}, nil
}

// updateStatus updates opstack node status
func (r *NodeReconciler) updateStatus(ctx context.Context, node *opstackv1alpha1.Node, enodeURL string) error {
	node.Status.Network = node.Spec.Network

	log := log.FromContext(ctx)

	if err := r.Status().Update(ctx, node); err != nil {
		log.Error(err, "unable to update node status")
		return err
	}

	return nil
}

// specPVC update node data pvc spec
func (r *NodeReconciler) specPVC(node *opstackv1alpha1.Node, pvc *corev1.PersistentVolumeClaim) {
	request := corev1.ResourceList{
		corev1.ResourceStorage: resource.MustParse(node.Spec.Resources.Storage),
	}

	// spec is immutable after creation except resources.requests for bound claims
	if !pvc.CreationTimestamp.IsZero() {
		pvc.Spec.Resources.Requests = request
		return
	}

	pvc.Labels = node.GetLabels()
	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.VolumeResourceRequirements{
			Requests: request,
		},
	}
}

// reconcilePVC reconciles node data persistent volume claim
func (r *NodeReconciler) reconcilePVC(ctx context.Context, node *opstackv1alpha1.Node) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: node.Namespace,
		},
	}

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if err := ctrl.SetControllerReference(node, pvc, r.Scheme); err != nil {
			return err
		}

		r.specPVC(node, pvc)

		return nil
	})

	return err
}

// createNodeVolumes creates node volumes
func (r *NodeReconciler) createNodeVolumes(node *opstackv1alpha1.Node) (volumes []corev1.Volume) {

	dataVolume := corev1.Volume{
		Name: "data",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: node.Name,
			},
		},
	}
	volumes = append(volumes, dataVolume)

	var projections []corev1.VolumeProjection

	// authenticated APIs jwt secret
	if node.Spec.JWTSecretName != "" {
		jwtSecretProjection := corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: node.Spec.JWTSecretName,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "secret",
						Path: "jwt.secret",
					},
				},
			},
		}
		projections = append(projections, jwtSecretProjection)
	}

	if node.Spec.JWTSecretName != "" {
		secretsVolume := corev1.Volume{
			Name: "secrets",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: projections,
				},
			},
		}
		volumes = append(volumes, secretsVolume)
	}

	return
}

// createNodeVolumeMounts creates all required volume mounts for the node
func (r *NodeReconciler) createNodeVolumeMounts(node *opstackv1alpha1.Node, homedir string) []corev1.VolumeMount {

	volumeMounts := []corev1.VolumeMount{}

	if node.Spec.JWTSecretName != "" {
		secretsMount := corev1.VolumeMount{
			Name:      "secrets",
			MountPath: shared.PathSecrets(homedir),
			ReadOnly:  true,
		}
		volumeMounts = append(volumeMounts, secretsMount)
	}

	dataMount := corev1.VolumeMount{
		Name:      "data",
		MountPath: shared.PathData(homedir),
	}
	volumeMounts = append(volumeMounts, dataMount)

	return volumeMounts
}

// specStatefulset updates node statefulset spec
func (r *NodeReconciler) specStatefulset(node *opstackv1alpha1.Node, sts *appsv1.StatefulSet, homedir string, args []string, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	labels := node.GetLabels()
	initContainers := []corev1.Container{}

	ports := []corev1.ContainerPort{
		{
			Name:          "discovery",
			ContainerPort: int32(node.Spec.P2PPort),
			Protocol:      corev1.ProtocolUDP,
		},
		{
			Name:          "p2p",
			ContainerPort: int32(node.Spec.P2PPort),
		},
	}

	if node.Spec.RPC {
		ports = append(ports, corev1.ContainerPort{
			Name:          "rpc",
			ContainerPort: int32(node.Spec.RPCPort),
		})
	}

	if node.Spec.WS {
		ports = append(ports, corev1.ContainerPort{
			Name:          "ws",
			ContainerPort: int32(node.Spec.WSPort),
		})
	}

	ports = append(ports, corev1.ContainerPort{
		Name:          "engine",
		ContainerPort: int32(node.Spec.EnginePort),
	})

	// execution client container
	nodeContainer := corev1.Container{
		Name:  "execution",
		Image: node.Spec.Image,
		Args:  args,
		Ports: ports,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(node.Spec.Resources.CPU),
				corev1.ResourceMemory: resource.MustParse(node.Spec.Resources.Memory),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(node.Spec.Resources.CPULimit),
				corev1.ResourceMemory: resource.MustParse(node.Spec.Resources.MemoryLimit),
			},
		},
		VolumeMounts: volumeMounts,
	}

	// op-node sidecar container
	opNodeArgs := []string{
		fmt.Sprintf("--l1=%s", node.Spec.L1Endpoint),
		fmt.Sprintf("--l1.beacon=%s", node.Spec.L1BeaconEndpoint),
		fmt.Sprintf("--network=%s", node.Spec.Network),
		fmt.Sprintf("--l2=http://127.0.0.1:%d", node.Spec.EnginePort),
		fmt.Sprintf("--l2.jwt-secret=%s/jwt.secret", shared.PathSecrets(homedir)),
		"--rpc.addr=0.0.0.0",
		"--rpc.port=9545",
	}

	opNodeArgs = append(opNodeArgs, node.Spec.NodeExtraArgs.Encode(false)...)

	opNodeContainer := corev1.Container{
		Name:  "op-node",
		Image: node.Spec.NodeImage,
		Args:  opNodeArgs,
		Ports: []corev1.ContainerPort{
			{
				Name:          "rollup-rpc",
				ContainerPort: 9545,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
		VolumeMounts: volumeMounts,
	}

	sts.ObjectMeta.Labels = labels
	if sts.Spec.Selector == nil {
		sts.Spec.Selector = &metav1.LabelSelector{}
	}

	replicas := int32(*node.Spec.Replicas)

	sts.Spec.Replicas = &replicas
	sts.Spec.ServiceName = node.Name
	sts.Spec.Selector.MatchLabels = labels
	sts.Spec.Template.ObjectMeta.Labels = labels
	sts.Spec.Template.Spec = corev1.PodSpec{
		SecurityContext: shared.SecurityContext(),
		Volumes:         volumes,
		InitContainers:  initContainers,
		Containers:      []corev1.Container{nodeContainer, opNodeContainer},
	}
}

// reconcileStatefulSet creates node statefulset if it doesn't exist, update it if it does exist
func (r *NodeReconciler) reconcileStatefulSet(ctx context.Context, node *opstackv1alpha1.Node) error {

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: node.Namespace,
		},
	}

	homedir := "/root"
	// Build basic args for execution clients
	args := []string{}
	switch node.Spec.Client {
	case opstackv1alpha1.OpRethClient, opstackv1alpha1.BaseRethNodeClient:
		args = append(args, "node", 
			fmt.Sprintf("--chain=%s", node.Spec.Network),
			fmt.Sprintf("--datadir=%s", shared.PathData(homedir)),
			fmt.Sprintf("--port=%d", node.Spec.P2PPort),
			"--discovery.port=30303",
			fmt.Sprintf("--authrpc.port=%d", node.Spec.EnginePort),
			"--authrpc.addr=0.0.0.0",
			fmt.Sprintf("--authrpc.jwtsecret=%s/jwt.secret", shared.PathSecrets(homedir)),
		)
		if node.Spec.RPC {
			args = append(args, "--http", "--http.addr=0.0.0.0", fmt.Sprintf("--http.port=%d", node.Spec.RPCPort))
		}
		if node.Spec.WS {
			args = append(args, "--ws", "--ws.addr=0.0.0.0", fmt.Sprintf("--ws.port=%d", node.Spec.WSPort))
		}
	case opstackv1alpha1.OpGethClient:
		args = append(args, 
			fmt.Sprintf("--networkid=%s", node.Spec.Network),
			fmt.Sprintf("--datadir=%s", shared.PathData(homedir)),
			fmt.Sprintf("--port=%d", node.Spec.P2PPort),
			fmt.Sprintf("--authrpc.port=%d", node.Spec.EnginePort),
			"--authrpc.addr=0.0.0.0",
			fmt.Sprintf("--authrpc.jwtsecret=%s/jwt.secret", shared.PathSecrets(homedir)),
			"--rollup.disabletxpoolgossip=true",
		)
		if node.Spec.RPC {
			args = append(args, "--http", "--http.addr=0.0.0.0", fmt.Sprintf("--http.port=%d", node.Spec.RPCPort))
		}
		if node.Spec.WS {
			args = append(args, "--ws", "--ws.addr=0.0.0.0", fmt.Sprintf("--ws.port=%d", node.Spec.WSPort))
		}
	}
	
	args = append(args, node.Spec.ExtraArgs.Encode(false)...)
	volumes := r.createNodeVolumes(node)
	mounts := r.createNodeVolumeMounts(node, homedir)

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, sts, func() error {
		if err := ctrl.SetControllerReference(node, sts, r.Scheme); err != nil {
			return err
		}
		r.specStatefulset(node, sts, homedir, args, volumes, mounts)
		return nil
	})

	return err
}

// generateJWTSecret generates a 32 byte hex-encoded random string
func generateJWTSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// specSecret creates jwt secret
func (r *NodeReconciler) specSecret(ctx context.Context, node *opstackv1alpha1.Node, secret *corev1.Secret) error {
	secret.ObjectMeta.Labels = node.GetLabels()
	
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	// generate if doesn't exist
	if len(secret.Data["secret"]) == 0 {
		jwtSecret := generateJWTSecret()
		secret.Data["secret"] = []byte(jwtSecret)
	}

	return nil
}

// reconcileSecret creates node secret if it doesn't exist, update it if it exists
func (r *NodeReconciler) reconcileSecret(ctx context.Context, node *opstackv1alpha1.Node) (jwtSecretName string, err error) {
	jwtSecretName = fmt.Sprintf("%s-jwt", node.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jwtSecretName,
			Namespace: node.Namespace,
		},
	}

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if err := ctrl.SetControllerReference(node, secret, r.Scheme); err != nil {
			return err
		}

		return r.specSecret(ctx, node, secret)
	})

	return
}

// specService updates node service spec
func (r *NodeReconciler) specService(node *opstackv1alpha1.Node, svc *corev1.Service) {
	labels := node.GetLabels()

	svc.ObjectMeta.Labels = labels
	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "discovery",
			Port:       int32(node.Spec.P2PPort),
			TargetPort: intstr.FromString("discovery"),
			Protocol:   corev1.ProtocolUDP,
		},
		{
			Name:       "p2p",
			Port:       int32(node.Spec.P2PPort),
			TargetPort: intstr.FromString("p2p"),
			Protocol:   corev1.ProtocolTCP,
		},
		{
			Name:       "rollup-rpc",
			Port:       9545,
			TargetPort: intstr.FromInt(9545),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	if node.Spec.RPC {
		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name:       "rpc",
			Port:       int32(node.Spec.RPCPort),
			TargetPort: intstr.FromString("rpc"),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	if node.Spec.WS {
		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name:       "ws",
			Port:       int32(node.Spec.WSPort),
			TargetPort: intstr.FromString("ws"),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	// Engine API is always required for OP Stack
	svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
		Name:       "engine",
		Port:       int32(node.Spec.EnginePort),
		TargetPort: intstr.FromString("engine"),
		Protocol:   corev1.ProtocolTCP,
	})

	svc.Spec.Selector = labels
}

// reconcileService reconciles node service
func (r *NodeReconciler) reconcileService(ctx context.Context, node *opstackv1alpha1.Node) (ip string, err error) {

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Name,
			Namespace: node.Namespace,
		},
	}

	_, err = ctrl.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(node, svc, r.Scheme); err != nil {
			return err
		}

		r.specService(node, svc)

		return nil
	})

	if err != nil {
		return
	}

	ip = svc.Spec.ClusterIP

	return
}

// SetupWithManager adds reconciler to the manager
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&opstackv1alpha1.Node{}).
		WithEventFilter(pred).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
