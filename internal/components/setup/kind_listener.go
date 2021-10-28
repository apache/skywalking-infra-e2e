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

package setup

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// KindContainerListener listen or get all kubernetes pod
type KindContainerListener struct {
	clientGetter *util.K8sClusterInfo
	ctx          context.Context
	ctxCancel    context.CancelFunc
}

func NewKindContainerListener(ctx context.Context, clientGetter *util.K8sClusterInfo) *KindContainerListener {
	childCtx, cancelFunc := context.WithCancel(ctx)
	return &KindContainerListener{
		clientGetter: clientGetter,
		ctx:          childCtx,
		ctxCancel:    cancelFunc,
	}
}

// Listen pod event
func (c *KindContainerListener) Listen(consumer func(pod *v1.Pod)) error {
	result := c.buildSearchResult()

	runtimeObject, err := result.Object()
	if err != nil {
		return err
	}
	watchVersion, err := meta.NewAccessor().ResourceVersion(runtimeObject)
	if err != nil {
		return err
	}

	watcher, err := result.Watch(watchVersion)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.ResultChan():
				switch event.Type {
				case watch.Added, watch.Modified:
					pod, err := c.unstructuredToPod(event.Object.(*unstructured.Unstructured))
					if err != nil {
						continue
					}
					consumer(pod)
				case watch.Error:
					errObject := apierrors.FromObject(event.Object)
					statusErr := errObject.(*apierrors.StatusError)
					logger.Log.Warnf("watch kubernetes pod error, %v", statusErr)
				}
			case <-c.ctx.Done():
				watcher.Stop()
				c.ctxCancel()
				return
			}
		}
	}()

	return nil
}

func (c *KindContainerListener) GetAllPods() ([]*v1.Pod, error) {
	result := c.buildSearchResult()
	infos, err := result.Infos()
	if err != nil {
		return nil, err
	}

	pods := make([]*v1.Pod, 0)
	for _, info := range infos {
		pod, err := c.unstructuredToPod(info.Object.(*unstructured.Unstructured))
		if err != nil {
			return nil, err
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func (c *KindContainerListener) Stop() {
	c.ctxCancel()
}

func (c *KindContainerListener) buildSearchResult() *resource.Result {
	return resource.NewBuilder(c.clientGetter).
		Unstructured().
		AllNamespaces(true).
		ResourceTypeOrNameArgs(true, "pods").
		Latest().
		Flatten().
		Do()
}

func (c *KindContainerListener) unstructuredToPod(object *unstructured.Unstructured) (*v1.Pod, error) {
	var pod v1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.UnstructuredContent(), &pod); err != nil {
		return nil, err
	}
	return &pod, nil
}
