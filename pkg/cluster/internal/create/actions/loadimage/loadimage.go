/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package waitforready implements the wait for ready action
package loadimage

import (
	"fmt"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

// Action implements an action for load image to specified node
type Action struct{}

// NewAction returns a new action for load image to specified node
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("load image to specified node ")
	defer ctx.Status.End(false)
	nodeList, err := ctx.Provider.ListNodes(ctx.Config.Name)
	if err != nil {
		return err
	}
	if len(nodeList) == 0 {
		return fmt.Errorf("no nodes found for cluster %q", ctx.Config.Name)
	}
	// map cluster nodes by their name
	nodesByName := map[string]nodes.Node{}
	for _, node := range nodeList {
		// TODO(bentheelder): this depends on the fact that ListByCluster()
		// will have name for nameOrId.
		nodesByName[node.String()] = node
	}

	// create node load image
	loadImage := func(images []string, nodeList []nodes.Node) func() error {
		return func() error {
			return ctx.Provider.LoadImage(images, nodeList)
		}
	}
	fns := []func() error{}
	nodeNamer := common.MakeNodeNamer(ctx.Config.Name)
	for _, nodeCfg := range ctx.Config.Nodes {
		nodeName := nodeNamer(string(nodeCfg.Role))
		imageNames := common.RemoveDuplicates(nodeCfg.LoadImages)
		if len(imageNames) == 0 {
			continue
		}
		if node, ok := nodesByName[nodeName]; ok {
			fns = append(fns, loadImage(imageNames, []nodes.Node{node}))
		}
	}
	if err := errors.UntilErrorConcurrent(fns); err != nil {
		return err
	}

	// mark success
	ctx.Status.End(true)
	ctx.Logger.V(0).Infof(" load succefully 💚")
	return nil
}
