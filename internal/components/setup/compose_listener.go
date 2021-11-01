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

	"github.com/docker/docker/api/types/events"

	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type ComposeContainerListener struct {
	client   *client.Client
	services []*ComposeService
	ctx      context.Context
	cancel   context.CancelFunc
}

type ComposeContainer struct {
	Service *ComposeService
	ID      string
}

func NewComposeContainerListener(ctx context.Context, cli *client.Client, services []*ComposeService) *ComposeContainerListener {
	childCtx, cancelFunc := context.WithCancel(ctx)
	return &ComposeContainerListener{
		client:   cli,
		services: services,
		ctx:      childCtx,
		cancel:   cancelFunc,
	}
}

func (c *ComposeContainerListener) Listen(consumer func(container *ComposeContainer)) error {
	containerEvents, errors := c.client.Events(c.ctx, types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
		),
	})

	if len(errors) > 0 {
		return <-errors
	}

	go func() {
		for {
			select {
			case msg := <-containerEvents:
				container := c.foundMessage(&msg)
				if container != nil {
					consumer(container)
				}
			case err := <-errors:
				if err != nil {
					logger.Log.Warnf("Listen docker container failed, %v", err)
				}
			case <-c.ctx.Done():
				c.cancel()
				return
			}
		}
	}()
	return nil
}

func (c *ComposeContainerListener) Stop() {
	c.cancel()
}

func (c *ComposeContainerListener) foundMessage(message *events.Message) *ComposeContainer {
	serviceName := message.Actor.Attributes["com.docker.compose.service"]
	for _, service := range c.services {
		if service.Name == serviceName {
			return &ComposeContainer{
				Service: service,
				ID:      message.ID,
			}
		}
	}
	return nil
}
