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
	"os"
	"path/filepath"
	"strings"

	apiv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

// K8sClusterInfo created when connect to cluster
type K8sClusterInfo struct {
	Client     *kubernetes.Clientset
	Interface  dynamic.Interface
	restConfig *rest.Config
	namespace  string
}

// ConnectToK8sCluster gets clientSet and dynamic client from k8s config file.
func ConnectToK8sCluster(kubeConfigPath string) (info *K8sClusterInfo, err error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	kubeConfigYaml, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigYaml)
	if err != nil {
		return nil, err
	}

	logger.Log.Info("connect to k8s cluster succeeded")

	return &K8sClusterInfo{c, dc, restConfig, ""}, nil
}

func (c *K8sClusterInfo) CopyClusterToNamespace(namespace string) *K8sClusterInfo {
	return &K8sClusterInfo{
		Client:     c.Client,
		Interface:  c.Interface,
		restConfig: c.restConfig,
		namespace:  namespace,
	}
}

func (c *K8sClusterInfo) ToRESTConfig() (*rest.Config, error) {
	return c.restConfig, nil
}

func (c *K8sClusterInfo) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	config.Burst = 100

	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(config)
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (c *K8sClusterInfo) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

func (c *K8sClusterInfo) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	overrides.Context.Namespace = c.namespace

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
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
				s = append(s, f)
			}
		}
	}
	return s, nil
}

// OperateManifest operates manifest in k8s cluster which kind created.
func OperateManifest(c *kubernetes.Clientset, dc dynamic.Interface, manifest string, operation apiv1.Operation) error {
	b, err := os.ReadFile(manifest)
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
