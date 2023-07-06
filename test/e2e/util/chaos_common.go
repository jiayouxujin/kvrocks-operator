package util

import (
	"context"
	"fmt"
	"time"

	chaosmeshv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ChaosMeshNamespace = "chaos-mesh"
)

type Experiment struct {
	chaosObject client.Object
	name        string
	namespace   string
}

func (env *KubernetesEnv) addChaosExperiment(experiment Experiment) {
	env.ChaosMeshExperiments = append(env.ChaosMeshExperiments, experiment)
}

func (env *KubernetesEnv) CreateExperiment(chaos client.Object) *Experiment {
	fmt.Fprintf(GinkgoWriter, "CreateExperiment name=%s\n", chaos.GetName())
	err := env.Client.Create(context.Background(), chaos)
	Expect(err).NotTo(HaveOccurred())

	// create chaos experiment
	experiment := Experiment{
		chaosObject: chaos,
		name:        chaos.GetName(),
		namespace:   chaos.GetNamespace(),
	}

	env.addChaosExperiment(experiment)

	return &experiment
}

func (env *KubernetesEnv) waitUnitlExperimentRunning(experiment Experiment, out client.Object) error {
	err := wait.PollImmediate(30*time.Second, 10*time.Minute, func() (bool, error) {
		err := env.Client.Get(context.Background(), client.ObjectKey{Name: experiment.name, Namespace: experiment.namespace}, out)
		if err != nil {
			return false, nil
		}

		return isRunning(out)
	})
	if err != nil {
		experiment.chaosObject = out
		return fmt.Errorf("waitUnitlExperimentRunning failed: %v", err)
	}
	return nil
}

func isRunning(obj client.Object) (bool, error) {
	podChaos, ok := obj.(*chaosmeshv1alpha1.PodChaos)
	if ok {
		return conditionsAreTrue(podChaos.GetStatus(), podChaos.GetStatus().Conditions), nil
	}
	return false, fmt.Errorf("isRunning failed: %v", obj)
}

func conditionsAreTrue(status *chaosmeshv1alpha1.ChaosStatus, conditions []chaosmeshv1alpha1.ChaosCondition) bool {
	var allInjected, allSelected bool

	for _, condition := range conditions {
		if condition.Type == chaosmeshv1alpha1.ConditionAllInjected {
			allInjected = condition.Status == corev1.ConditionTrue
		}

		if condition.Type == chaosmeshv1alpha1.ConditionSelected {
			allSelected = condition.Status == corev1.ConditionTrue
		}
	}

	fmt.Println(
		"experiment conditions - allInjected:",
		allInjected,
		"allSelected:",
		allSelected,
		"status",
		status,
		"count records",
		len(status.Experiment.Records),
	)

	for _, stat := range status.Experiment.Records {
		fmt.Println("Records stat ID", stat.Id, "phase:", stat.Phase, "selector", stat.SelectorKey)
	}

	return allInjected && allSelected
}
