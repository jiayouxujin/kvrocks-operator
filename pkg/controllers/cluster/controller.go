package cluster

import "github.com/RocksLabs/kvrocks-operator/pkg/resources"

func (h *KVRocksClusterHandler) ensureController() error {
	etcd := resources.NewEtcdStatefulSet(h.instance)
	if err := h.k8s.CreateIfNotExistsOriginalStatefulSet(etcd); err != nil {
		return err
	}
	etcdService := resources.NewEtcdService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(etcdService); err != nil {
		return err
	}
	controllerConfigmap := resources.NewKVRocksControllerConfigmap()
	if err := h.k8s.CreateIfNotExistsConfigMap(controllerConfigmap); err != nil {
		return err
	}
	controllerService := resources.NewKVRocksControllerService(h.instance)
	if err := h.k8s.CreateIfNotExistsService(controllerService); err != nil {
		return err
	}
	controllerDep := resources.NewKVRocksControllerDeployment(h.instance)
	if err := h.k8s.CreateIfNotExistsDeployment(controllerDep); err != nil {
		return err
	}
	return nil
}
