// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package util

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	apiv1 "k8s.io/api/admission/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

// ConnectToK8sCluster gets clientSet and dynamic client from k8s config file.
func ConnectToK8sCluster(kubeConfigPath string) (c *kubernetes.Clientset, dc dynamic.Interface, err error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, nil, err
	}
	c, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	dc, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	logger.Log.Info("connect to k8s cluster succeeded")

	return c, dc, nil
}

// GetManifests recursively gets all yml and yaml files from manifests string.
func GetManifests(manifests string) (files []string, err error) {
	s := make([]string, 0)
	files = strings.Split(manifests, ",")
	// file or directory
	for _, f := range files {
		f = ResolveAbs(f)
		fi, err := os.Stat(f)
		if err != nil {
			return nil, err
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			err := filepath.Walk(f, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
					path = ResolveAbs(path)
					s = append(s, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		case mode.IsRegular():
			filename := fi.Name()
			if strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml") {
				filename = ResolveAbs(filename)
				s = append(s, filename)
			}
		}
	}
	return s, nil
}

// OperateManifest operates manifest in k8s cluster which kind created.
func OperateManifest(c *kubernetes.Clientset, dc dynamic.Interface, manifest string, operation apiv1.Operation) error {
	b, err := ioutil.ReadFile(manifest)
	if err != nil {
		return err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
		apiGroupResource, err := restmapper.GetAPIGroupResources(c.Discovery())
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(apiGroupResource)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// constrict resources to the default namespace
			if unstructuredObj.GetNamespace() != "" && unstructuredObj.GetNamespace() != metav1.NamespaceDefault {
				return fmt.Errorf("all resources must in default namespace")
			}
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(metav1.NamespaceDefault)
			}
			dri = dc.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dc.Resource(mapping.Resource)
		}

		switch operation {
		case apiv1.Create:
			_, err = dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		case apiv1.Delete:
			err = dri.Delete(context.Background(), unstructuredObj.GetName(), metav1.DeleteOptions{})
		}

		if err != nil {
			return err
		}
	}

	return nil
}
