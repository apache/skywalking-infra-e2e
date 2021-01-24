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
	"fmt"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/util"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

// waitForPodsReady waits all pods to be ready.
func waitForPodsReady(c *kubernetes.Clientset, waitSet *util.WaitSet, namespace, selector string) {
	defer waitSet.WaitGroup.Done()

	podList, err := listAllPods(c, namespace, selector)
	if err != nil {
		err = fmt.Errorf("list all pods error. namespce: %s selector: %s error: %s", namespace, selector, err)
		waitSet.ErrChan <- err
		return
	}

	if len(podList.Items) == 0 {
		waitSet.ErrChan <- fmt.Errorf("no pods can be wait. namespce: %s selector: %s", namespace, selector)
		return
	}

	// use idx instead of object itself to avoid rangeValCopy in gocritic
	for idx := range podList.Items {
		pod := podList.Items[idx]
		logger.Log.Infof("waiting for pod %s ready...", pod.Name)
		if err := waitForOnePodReady(c, pod.Namespace, pod.Name); err != nil {
			err = fmt.Errorf("wait for pod %s error. namespace: %s error: %s", pod.Name, pod.Namespace, err)
			waitSet.ErrChan <- err
			return
		}
		logger.Log.Infof("pod %s ready", pod.Name)
	}
}

func listAllPods(c *kubernetes.Clientset, namespace, selector string) (list *v1.PodList, err error) {
	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}
	ctx := context.TODO()
	podList, err := c.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func waitForOnePodReady(c *kubernetes.Clientset, namespace, podName string) error {
	return wait.PollInfinite(time.Second, isPodReady(c, podName, namespace))
}

func isPodReady(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		ctx := context.TODO()
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		podConditions := pod.Status.Conditions
		for _, condition := range podConditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true, nil
			}
		}

		if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
			return false, fmt.Errorf("pod ran to completion")
		}
		return false, nil
	}
}
