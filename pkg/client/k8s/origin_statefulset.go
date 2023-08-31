package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) CreateIfNotExistsOriginalStatefulSet(sts *appsv1.StatefulSet) error {
	if err := c.client.Create(ctx, sts); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	c.logger.V(1).Info("create statefulSet successfully", "statefulSet", sts.Name)
	return nil
}


func (c *Client) GetOriginalStatefulSet(key types.NamespacedName) (*appsv1.StatefulSet, error) {
	var sts appsv1.StatefulSet
	if err := c.client.Get(ctx, key, &sts); err != nil {
		return nil, err
	}
	return &sts, nil
}