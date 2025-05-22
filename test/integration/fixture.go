//
// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer" // Standard Kubernetes scheme

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateObjectsFromYAMLFiles reads YAML files, decodes them into Kubernetes objects,
// and creates them using the provided client.
// It handles multi-document YAML files (separated by "---").
func CreateObjectsFromYAMLFiles(ctx context.Context, k8sClient ctrlruntimeclient.Client, namespace, dir string) error {
	logger := log.FromContext(ctx)

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, filePath := range files {
		if filePath.IsDir() {
			continue // Skip directories
		}
		fullPath := filepath.Join(dir, filePath.Name())
		logger.Info("Processing file", "path", filePath)

		obj, gvk, err := RuntimeFromYaml(k8sClient, fullPath)
		if err != nil {
			return err
		}

		clientObj, ok := obj.(ctrlruntimeclient.Object)
		if !ok {
			err := fmt.Errorf("decoded object from embedded file is not a client.Object: %T (GVK: %s)", obj, gvk.String())
			logger.Error(err, "Type assertion failed", "path", filePath)
			return err
		}
		clientObj.SetNamespace(namespace)
		if err := k8sClient.Create(ctx, clientObj); err != nil {
			logger.Error(err, "Failed to create object in cluster", "details", ctrlruntimeclient.ObjectKeyFromObject(clientObj))
			return fmt.Errorf("failed to create object %s: %w", ctrlruntimeclient.ObjectKeyFromObject(clientObj), err)
		}
	}
	return nil
}

// RuntimeFromYaml returns a list of Kubernetes runtime objects from their yaml templates.
func RuntimeFromYaml(ctrlclient ctrlruntimeclient.Client, filePath string) (runtime.Object, *schema.GroupVersionKind, error) {
	decode := serializer.NewCodecFactory(ctrlclient.Scheme()).UniversalDeserializer().Decode

	manifest, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	obj, gvk, err := decode(manifest, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	return obj, gvk, nil
}
