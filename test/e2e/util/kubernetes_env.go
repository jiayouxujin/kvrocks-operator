package util

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kvrocksv1alpha1 "github.com/RocksLabs/kvrocks-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesEnv struct {
	config               *Config
	kubernetesConfig     *rest.Config
	Client               client.Client
	ChaosMeshExperiments []Experiment
	Clean                func() error
}

func Start(config *Config) *KubernetesEnv {
	env := &KubernetesEnv{
		config: config,
		Clean: func() error {
			return nil
		},
	}

	if env.config.KubeConfig == "" {
		env.installKubernetes()
		env.config.KubeConfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		env.Clean = func() error {
			fmt.Fprintf(GinkgoWriter, "uninstall kubernetes cluster %s\n", env.config.ClusterName)
			cmd := exec.Command(getClusterInstallScriptPath(), "down", "-c", env.config.ClusterName)
			err := cmd.Run()
			return err
		}
		env.localToCluster()
	}

	// scheme
	env.registerScheme()

	//config
	cfg, err := loadKubernetesConfig(env.config.KubeConfig)
	Expect(err).NotTo(HaveOccurred())
	env.kubernetesConfig = cfg

	env.Client, err = client.New(env.kubernetesConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	env.installKruise()
	env.installKvrocksOperator()
	env.installChaosMesh()
	return env
}

func (env *KubernetesEnv) installKubernetes() {
	fmt.Fprintf(GinkgoWriter, "install kubernetes cluster %s\n", env.config.ClusterName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel() // This ensures resources are cleaned up.

	cmd := exec.CommandContext(ctx, getClusterInstallScriptPath(), "up", "-c", env.config.ClusterName)
	done := make(chan error)

	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		Fail("install kubernetes cluster timeout")
	case err := <-done:
		if err != nil {
			Fail(fmt.Sprintf("Error occurred: %v", err)) // If an error occurred running the command, report it.
		}
	}
}

// Using telepresence tool to connect to the kubernetes cluster
func (env *KubernetesEnv) localToCluster() {
	if _, err := exec.Command("telepresence", "helm", "install").CombinedOutput(); err != nil {
		Fail(fmt.Sprintf("error occurred when using telepresence to connect to the kubernetes cluster %s", err.Error()))
	}
	if _, err := exec.Command("telepresence", "connect").CombinedOutput(); err != nil {
		Fail(fmt.Sprintf("error occurred when using telepresence to connect to the kubernetes cluster %s", err.Error()))
	}
}

func (env *KubernetesEnv) registerScheme() {
	err := kvrocksv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = kruise.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
}

func (env *KubernetesEnv) installKruise() {
	fmt.Fprintf(GinkgoWriter, "install kruise %s\n", env.config.KruiseVersion)

	// Add OpenKruise Helm repo
	repoList, err := HelmTool("repo", "list")
	if err != nil && strings.Contains(err.Error(), "Error: no repositories to show") {
		err = nil
	}
	Expect(err).Should(Succeed())
	if !strings.Contains(repoList, "openkruise") {
		_, err := HelmTool("repo", "add", "openkruise", "https://openkruise.github.io/charts")
		Expect(err).NotTo(HaveOccurred())
	}

	// Update Helm repo
	_, err = HelmTool("repo", "update")
	Expect(err).NotTo(HaveOccurred())

	// Install OpenKruise using Helm
	if !env.isHelmInstalled("kruise", "default") {
		_, err = HelmTool("install", "kruise", "openkruise/kruise", "--version", env.config.KruiseVersion, "--wait")
		Expect(err).NotTo(HaveOccurred())
	}
}

func (env *KubernetesEnv) installKvrocksOperator() {
	fmt.Fprintf(GinkgoWriter, "install kvrocks operator\n")

	if !env.isExistsCRD("kvrocks.kvrocks.com") {
		_, err := KubectlTool("apply", "-f", "../../../deploy/crd/templates/crd.yaml")
		Expect(err).NotTo(HaveOccurred())
	}

	if !env.isHelmInstalled("kvrocks-operator", env.config.Namespace) {
		_, err := HelmTool("install", "kvrocks-operator", "../../../deploy/operator", "-n", env.config.Namespace, "--create-namespace")
		Expect(err).NotTo(HaveOccurred())
	}
}

func (env *KubernetesEnv) installChaosMesh() {
	if env.config.ChaosMeshEnabled {
		fmt.Fprintf(GinkgoWriter, "install chaosmesh\n")

		// Add Chaosmesh Helm repo
		repoList, err := HelmTool("repo", "list")
		if err != nil && strings.Contains(err.Error(), "Error: no repositories to show") {
			err = nil
		}
		Expect(err).Should(Succeed())
		if !strings.Contains(repoList, "chaos-mesh") {
			_, err := HelmTool("repo", "add", "chaos-mesh", "https://charts.chaos-mesh.org")
			Expect(err).NotTo(HaveOccurred())
		}

		// Update Helm repo
		_, err = HelmTool("repo", "update")
		Expect(err).NotTo(HaveOccurred())

		// Create ns
		_, err = KubectlTool("create", "ns", "chaos-mesh")
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())

		// Install OpenKruise using Helm
		if !env.isHelmInstalled("chaos-mesh", "chaos-mesh") {
			_, err = HelmTool("install", "chaos-mesh", "chaos-mesh/chaos-mesh", "-n=chaos-mesh", "--wait")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func (env *KubernetesEnv) isExistsCRD(name string) bool {
	_, err := KubectlTool("get", "crd", name)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			Expect(err).NotTo(HaveOccurred())
		}
		return false
	}
	return true
}

func (env *KubernetesEnv) isHelmInstalled(name string, namespace string) bool {
	_, err := HelmTool("status", name, "-n", namespace)
	if err != nil {
		if !strings.Contains(err.Error(), "release: not found") {
			Expect(err).NotTo(HaveOccurred())
		}
		return false
	}
	return true
}

func loadKubernetesConfig(kubeConfig string) (*rest.Config, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func getClusterInstallScriptPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filepath.Dir(currentFile)), "script", DefaultMinikubeShell)
}
