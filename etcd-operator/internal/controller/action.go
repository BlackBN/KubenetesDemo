package controller

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action interface {
	Execute(ctx context.Context) error
}

type PatchStatus struct {
	client client.Client
	oldObj client.Object
	newObj client.Object
}

func (p *PatchStatus) Execute(ctx context.Context) error {
	if reflect.DeepEqual(p.oldObj, p.newObj) {
		return nil
	}
	if err := p.client.Status().Patch(ctx, p.newObj, client.MergeFrom(p.oldObj)); err != nil {
		return fmt.Errorf("patching status error: %v", err)
	}
	return nil
}

type CreateObject struct {
	client client.Client
	obj    client.Object
}

func (c *CreateObject) Execute(ctx context.Context) error {
	if err := c.client.Create(ctx, c.obj); err != nil {
		return fmt.Errorf("create obj error: %v", err)
	}
	return nil
}
