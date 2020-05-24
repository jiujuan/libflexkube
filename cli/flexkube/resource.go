package flexkube

import (
	"fmt"
	"io/ioutil"

	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"

	flexcli "github.com/flexkube/libflexkube/cli"
	"github.com/flexkube/libflexkube/pkg/apiloadbalancer"
	"github.com/flexkube/libflexkube/pkg/container"
	"github.com/flexkube/libflexkube/pkg/controlplane"
	"github.com/flexkube/libflexkube/pkg/etcd"
	"github.com/flexkube/libflexkube/pkg/kubelet"
	"github.com/flexkube/libflexkube/pkg/kubernetes/client"
	"github.com/flexkube/libflexkube/pkg/pki"
	"github.com/flexkube/libflexkube/pkg/types"
)

// Resource represents flexkube CLI configuration structure.
type Resource struct {
	Etcd                 *etcd.Cluster                                `json:"etcd,omitempty"`
	Controlplane         *controlplane.Controlplane                   `json:"controlplane,omitempty"`
	PKI                  *pki.PKI                                     `json:"pki,omitempty"`
	KubeletPools         map[string]*kubelet.Pool                     `json:"kubeletPools,omitempty"`
	APILoadBalancerPools map[string]*apiloadbalancer.APILoadBalancers `json:"apiLoadBalancerPools,omitempty"`
	State                *ResourceState                               `json:"state,omitempty"`
}

// ResourceState represents flexkube CLI state format.
type ResourceState struct {
	Etcd                 *container.ContainersState            `json:"etcd,omitempty"`
	Controlplane         *container.ContainersState            `json:"controlplane,omitempty"`
	KubeletPools         map[string]*container.ContainersState `json:"kubeletPools,omitempty"`
	APILoadBalancerPools map[string]*container.ContainersState `json:"apiLoadBalancerPools,omitempty"`
	PKI                  *pki.PKI                              `json:"pki,omitempty"`
}

// getEtcd returns etcd resource, with state and PKI integration enabled.
func (r *Resource) getEtcd() (types.Resource, error) {
	if r.Etcd == nil {
		return nil, fmt.Errorf("etcd management not enabled in the configuration")
	}

	if r.State != nil && r.State.Etcd != nil {
		r.Etcd.State = *r.State.Etcd
	}

	// Enable PKI integration.
	if r.State != nil && r.State.PKI != nil {
		r.Etcd.PKI = r.State.PKI
	}

	return validateAndNew(r.Etcd)
}

// getControlplane returns controlplane resource, with state and PKI integration enabled.
func (r *Resource) getControlplane() (types.Resource, error) {
	if r.Controlplane == nil {
		return nil, fmt.Errorf("controlplane not configured")
	}

	if r.State != nil {
		r.Controlplane.State = r.State.Controlplane
	}

	// Enable PKI integration.
	if r.State != nil && r.State.PKI != nil {
		r.Controlplane.PKI = r.State.PKI
	}

	return validateAndNew(r.Controlplane)
}

// getKubeletPool returns requested kubelet pool with state and PKI injected.
func (r *Resource) getKubeletPool(name string) (types.Resource, error) {
	pool, ok := r.KubeletPools[name]
	if !ok {
		return nil, fmt.Errorf("pool not configured")
	}

	if r.State != nil && r.State.KubeletPools != nil && r.State.KubeletPools[name] != nil {
		pool.State = *r.State.KubeletPools[name]
	}

	// Enable PKI integration.
	if r.State != nil && r.State.PKI != nil {
		pool.PKI = r.State.PKI
	}

	return validateAndNew(pool)
}

// getPKI returns PKI struct with state loaded on top.
func (r *Resource) getPKI() (*pki.PKI, error) {
	if r.PKI == nil {
		return nil, fmt.Errorf("PKI config configured")
	}

	pki := &pki.PKI{}

	// If state contains PKI, use it as a base for loading.
	if r.State != nil && r.State.PKI != nil {
		fmt.Println("Loading existing PKI state from state.yaml file")

		pki = r.State.PKI
	}

	// Then load config on top.
	pkic, err := yaml.Marshal(r.PKI)
	if err != nil {
		return nil, fmt.Errorf("serializing PKI configuration failed: %w", err)
	}

	if err := yaml.Unmarshal(pkic, pki); err != nil {
		return nil, fmt.Errorf("failed merging PKI configuration with state: %w", err)
	}

	return pki, nil
}

// getAPILoadBalancerPool returns requested kubelet pool with state injected.
func (r *Resource) getAPILoadBalancerPool(name string) (types.Resource, error) {
	pool, ok := r.APILoadBalancerPools[name]
	if !ok {
		return nil, fmt.Errorf("pool not configured")
	}

	if r.State != nil && r.State.APILoadBalancerPools != nil && r.State.APILoadBalancerPools[name] != nil {
		pool.State = *r.State.APILoadBalancerPools[name]
	}

	return validateAndNew(pool)
}

// validateAndNew validates and creates new resource from resource config.
func validateAndNew(rc types.ResourceConfig) (types.Resource, error) {
	if err := rc.Validate(); err != nil {
		return nil, fmt.Errorf("validating configuration failed: %w", err)
	}

	r, err := rc.New()
	if err != nil {
		return nil, fmt.Errorf("initializing object failed: %w", err)
	}

	return r, nil
}

// execute deploys given resource and persists generated state to disk.
func (r *Resource) execute(rs types.Resource, saveStateF func(types.Resource)) error {
	// Check current state.
	fmt.Println("Checking current state")

	if err := rs.CheckCurrentState(); err != nil {
		return fmt.Errorf("failed checking current state: %w", err)
	}

	deployErr := rs.Deploy()

	if r.State == nil {
		r.State = &ResourceState{}
	}

	saveStateF(rs)

	return r.StateToFile(deployErr)
}

// LoadResourceFromFiles loads Resource struct from config.yaml and state.yaml files.
func LoadResourceFromFiles() (*Resource, error) {
	fmt.Println("Trying to read config.yaml and state.yaml files...")

	r := &Resource{}

	c, err := flexcli.ReadYamlFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading config.yaml file failed: %w", err)
	}

	s, err := flexcli.ReadYamlFile("state.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading state.yaml file failed: %w", err)
	}

	if err := yaml.Unmarshal([]byte(string(c)+string(s)), r); err != nil {
		return nil, fmt.Errorf("parsing files failed: %w", err)
	}

	return r, nil
}

// StateToFile saves resource state into state.yaml file.
func (r *Resource) StateToFile(actionErr error) error {
	rs := &Resource{
		State: r.State,
	}

	rb, err := yaml.Marshal(rs)
	if err != nil {
		return fmt.Errorf("failed serializing state: %w", err)
	}

	if string(rb) == "{}\n" {
		rb = []byte{}
	}

	if err := ioutil.WriteFile("state.yaml", rb, 0600); err != nil {
		if actionErr == nil {
			return fmt.Errorf("failed writing new state to file: %w", err)
		}

		fmt.Println("Failed to write state.yaml file: %w", err)
	}

	if actionErr != nil {
		return fmt.Errorf("execution failed: %w", actionErr)
	}

	fmt.Println("Action complete")

	return nil
}

// validateKubeconfigPKI validates if required fields are populated in PKI field
// to generate admin kubeconfig file.
func (r *Resource) validateKubeconfigPKI() error {
	if r.State.PKI == nil {
		return fmt.Errorf("PKI management not enabled")
	}

	if r.State.PKI.Kubernetes == nil {
		return fmt.Errorf("Kubernetes PKI management not enabled") //nolint:stylecheck
	}

	if r.State.PKI.Kubernetes.AdminCertificate == nil {
		return fmt.Errorf("Kubernetes admin certificate not available in PKI") //nolint:stylecheck
	}

	return nil
}

// validateKubeconfigControlplane validates if required fields are populated in PKI field
// to generate admin kubeconfig file.
func (r *Resource) validateKubeconfigControlplane() error {
	if r.Controlplane == nil {
		return fmt.Errorf("Kubernetes controlplane management not enabled") //nolint:stylecheck
	}

	if r.Controlplane.APIServerAddress == "" {
		return fmt.Errorf("Kubernetes controlplane has no API server address set") //nolint:stylecheck
	}

	if r.Controlplane.APIServerPort == 0 {
		return fmt.Errorf("Kubernetes controlplane has no API server port set") //nolint:stylecheck
	}

	return nil
}

// validateKubeconfig validates, if kubeconfig content can be generated from current
// state of the resource.
func (r *Resource) validateKubeconfig() error {
	if err := r.validateKubeconfigPKI(); err != nil {
		return err
	}

	if err := r.validateKubeconfigControlplane(); err != nil {
		return err
	}

	return nil
}

// Kubeconfig generates content of kubeconfig file in YAML format from Controlplane and PKI
// configuration.
func (r *Resource) Kubeconfig() (string, error) {
	if err := r.validateKubeconfig(); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	cc := &client.Config{
		Server:            fmt.Sprintf("%s:%d", r.Controlplane.APIServerAddress, r.Controlplane.APIServerPort),
		CACertificate:     r.State.PKI.Kubernetes.CA.X509Certificate,
		ClientCertificate: r.State.PKI.Kubernetes.AdminCertificate.X509Certificate,
		ClientKey:         r.State.PKI.Kubernetes.AdminCertificate.PrivateKey,
	}

	k, err := cc.ToYAMLString()
	if err != nil {
		return "", fmt.Errorf("generating failed: %w", err)
	}

	return k, nil
}

// withResource is a helper for action functions.
func withResource(f func(*Resource) func(c *cli.Context) error) func(c *cli.Context) error {
	r, err := LoadResourceFromFiles()
	if err != nil {
		return func(c *cli.Context) error {
			return fmt.Errorf("reading configuration and state failed: %w", err)
		}
	}

	return f(r)
}

// RunAPILoadBalancerPool deploys given API Load Balancer pool.
func (r *Resource) RunAPILoadBalancerPool(name string) error {
	p, err := r.getAPILoadBalancerPool(name)
	if err != nil {
		return fmt.Errorf("failed getting API Load Balancer pool %q from configuration: %w", name, err)
	}

	saveStateF := func(rs types.Resource) {
		if r.State.APILoadBalancerPools == nil {
			r.State.APILoadBalancerPools = map[string]*container.ContainersState{}
		}

		r.State.APILoadBalancerPools[name] = &p.Containers().ToExported().PreviousState
	}

	return r.execute(p, saveStateF)
}

// RunControlplane deploys configured static controlplane.
func (r *Resource) RunControlplane() error {
	e, err := r.getControlplane()
	if err != nil {
		return fmt.Errorf("failed getting controlplane from the configuration: %w", err)
	}

	saveStateF := func(rs types.Resource) {
		r.State.Controlplane = &e.Containers().ToExported().PreviousState
	}

	return r.execute(e, saveStateF)
}

// RunEtcd deploys configured etcd cluster.
func (r *Resource) RunEtcd() error {
	e, err := r.getEtcd()
	if err != nil {
		return fmt.Errorf("preparing failed: %w", err)
	}

	saveStateF := func(rs types.Resource) {
		r.State.Etcd = &e.Containers().ToExported().PreviousState
	}

	return r.execute(e, saveStateF)
}

// RunKubeletPool deploys given kubelet pool.
func (r *Resource) RunKubeletPool(name string) error {
	p, err := r.getKubeletPool(name)
	if err != nil {
		return fmt.Errorf("failed getting kubelet pool %q from configuration: %w", name, err)
	}

	saveStateF := func(rs types.Resource) {
		if r.State.KubeletPools == nil {
			r.State.KubeletPools = map[string]*container.ContainersState{}
		}

		r.State.KubeletPools[name] = &p.Containers().ToExported().PreviousState
	}

	return r.execute(p, saveStateF)
}

// RunPKI generates configured PKI.
func (r *Resource) RunPKI() error {
	pki, err := r.getPKI()
	if err != nil {
		return fmt.Errorf("failed loading PKI configuration: %w", err)
	}

	fmt.Println("Generating PKI...")

	genErr := pki.Generate()

	if r.State == nil {
		r.State = &ResourceState{}
	}

	r.State.PKI = pki

	return r.StateToFile(genErr)
}