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
