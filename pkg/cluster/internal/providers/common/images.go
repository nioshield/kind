/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"os"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/sets"
)

// RequiredNodeImages returns the set of _node_ images specified by the config
// This does not include the loadbalancer image, and is only used to improve
// the UX by explicit pulling the node images prior to running
func RequiredNodeImages(cfg *config.Cluster) sets.String {
	images := sets.NewString()
	for _, node := range cfg.Nodes {
		images.Insert(node.Image)
	}
	return images
}

type (
	ImageTagFetcher func(nodes.Node, string) (map[string]bool, error)
)

// checkIfImageExists makes sure we only perform the reverse lookup of the ImageID to tag map
func CheckIfImageReTagRequired(node nodes.Node, imageID, imageName string, tagFetcher ImageTagFetcher) (exists, reTagRequired bool, sanitizedImage string) {
	tags, err := tagFetcher(node, imageID)
	if len(tags) == 0 || err != nil {
		exists = false
		return
	}
	exists = true
	sanitizedImage = sanitizeImage(imageName)
	if ok := tags[sanitizedImage]; ok {
		reTagRequired = false
		return
	}
	reTagRequired = true
	return
}

// loads an image tarball onto a node
func LoadImage(imageTarName string, node nodes.Node) error {
	f, err := os.Open(imageTarName)
	if err != nil {
		return errors.Wrap(err, "failed to open image")
	}
	defer f.Close()
	return nodeutils.LoadImageArchive(node, f)
}

// removeDuplicates removes duplicates from a string slice
func RemoveDuplicates(slice []string) []string {
	result := []string{}
	seenKeys := make(map[string]struct{})
	for _, k := range slice {
		if _, seen := seenKeys[k]; !seen {
			result = append(result, k)
			seenKeys[k] = struct{}{}
		}
	}
	return result
}

// sanitizeImage is a helper to return human readable image name
// This is a modified version of the same function found under providers/podman/images.go
func sanitizeImage(image string) (sanitizedName string) {
	const (
		defaultDomain    = "docker.io/"
		officialRepoName = "library"
	)
	sanitizedName = image

	if !strings.ContainsRune(image, '/') {
		sanitizedName = officialRepoName + "/" + image
	}

	i := strings.IndexRune(sanitizedName, '/')
	if i == -1 || (!strings.ContainsAny(sanitizedName[:i], ".:") && sanitizedName[:i] != "localhost") {
		sanitizedName = defaultDomain + sanitizedName
	}

	i = strings.IndexRune(sanitizedName, ':')
	if i == -1 {
		sanitizedName += ":latest"
	}

	return
}
