package cluster

import (
	"github.com/RocksLabs/kvrocks-operator/pkg/resources"
)

func (h *KVRocksClusterHandler) ensureController() error {
	etcdService := resources.NewEtcdService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(etcdService); err != nil {
		return err
	}
	etcd := resources.NewEtcdStatefulSet(h.instance)
	if err := h.k8s.CreateIfNotExistsOriginalStatefulSet(etcd); err != nil {
		return err
	}
	// get etcd
	etcd, err := h.k8s.GetOriginalStatefulSet(h.key)
	if err != nil {
		return err
	}
	if etcd.Status.ReadyReplicas != *etcd.Spec.Replicas {
		h.log.Info("waiting for etcd ready")
		h.requeue = true
		return nil
	}
	controllerConfigmap := resources.NewKVRocksControllerConfigmap(h.instance)
	if err := h.k8s.CreateIfNotExistsConfigMap(controllerConfigmap); err != nil {
		return err
	}
	controllerService := resources.NewKVRocksControllerService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(controllerService); err != nil {
		return err
	}
	// get service
	controllerService, err = h.k8s.GetService(h.key)
	if err != nil {
		return err
	}
	h.endpoint = "http://" + controllerService.Spec.ClusterIP + ":9379/api/v1/"
	controllerDep := resources.NewKVRocksControllerDeployment(h.instance)
	if err := h.k8s.CreateIfNotExistsDeployment(controllerDep); err != nil {
		return err
	}
	// get deployment
	controllerDep, err = h.k8s.GetDeployment(h.key)
	if err != nil {
		return err
	}
	if controllerDep.Status.ReadyReplicas != *controllerDep.Spec.Replicas {
		h.log.Info("waiting for controller deployment ready")
		h.requeue = true
		return nil
	}
	err = h.kvrocks.CreateNamespace(h.endpoint, "cluster-demo")
	if err != nil {
		return err
	}
	h.controllerNamespace = "cluster-demo"
	h.log.Info("controller is ready")
	return nil
}
