package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (c *Client) CreateIfNotExistsOriginalStatefulSet(sts *appsv1.StatefulSet) error {
	if err:=c.client.Create(ctx, sts); err!=nil && !errors.IsAlreadyExists(err){
		return err
	}
	c.logger.V(1).Info("create statefulSet successfully", "statefulSet", sts.Name)
	return nil
}
