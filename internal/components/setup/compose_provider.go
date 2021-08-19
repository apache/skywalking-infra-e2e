package setup

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	Bridge        = "bridge"         // Bridge network name (as well as driver)
	ReaperDefault = "reaper_default" // Default network name when bridge is not available

	TestcontainerLabel = "org.testcontainers.golang"
)

// NetworkRequest represents the parameters used to get a network
type NetworkRequest struct {
	Driver         string
	CheckDuplicate bool
	Internal       bool
	EnableIPv6     bool
	Name           string
	Labels         map[string]string
	Attachable     bool

	SkipReaper  bool   // indicates whether we skip setting up a reaper for this
	ReaperImage string //alternative reaper registry
}

type Log struct {
	LogType string
	Content []byte
}

type LogConsumer interface {
	Accept(Log)
}

type Network interface {
	Remove(context.Context) error // removes the network
}

// DockerContainer represents a container started using Docker
type DockerContainer struct {
	// Container ID from Docker
	ID         string
	WaitingFor wait.Strategy
	Image      string

	provider          *DockerProvider
	terminationSignal chan bool
	skipReaper        bool
	consumers         []LogConsumer
	raw               *types.ContainerJSON
	stopProducer      chan bool
}

func (c *DockerContainer) GetContainerID() string {
	return c.ID
}

// Endpoint gets proto://host:port string for the first exposed port
// Will returns just host:port if proto is ""
func (c *DockerContainer) Endpoint(ctx context.Context, proto string) (string, error) {
	ports, err := c.Ports(ctx)
	if err != nil {
		return "", err
	}

	// get first port
	var firstPort nat.Port
	for p := range ports {
		firstPort = p
		break
	}

	return c.PortEndpoint(ctx, firstPort, proto)
}

// PortEndpoint gets proto://host:port string for the given exposed port
// Will returns just host:port if proto is ""
func (c *DockerContainer) PortEndpoint(ctx context.Context, port nat.Port, proto string) (string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	outerPort, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	protoFull := ""
	if proto != "" {
		protoFull = fmt.Sprintf("%s://", proto)
	}

	return fmt.Sprintf("%s%s:%s", protoFull, host, outerPort.Port()), nil
}

// Host gets host (ip or name) of the docker daemon where the container port is exposed
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
// You can use the "TC_HOST" env variable to set this yourself
func (c *DockerContainer) Host(ctx context.Context) (string, error) {
	host, err := c.provider.daemonHost(ctx)
	if err != nil {
		return "", err
	}
	return host, nil
}

// MappedPort gets externally mapped port for a container port
func (c *DockerContainer) MappedPort(ctx context.Context, port nat.Port) (nat.Port, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return "", err
	}
	if inspect.ContainerJSONBase.HostConfig.NetworkMode == "host" {
		return port, nil
	}
	ports, err := c.Ports(ctx)
	if err != nil {
		return "", err
	}

	for k, p := range ports {
		if k.Port() != port.Port() {
			continue
		}
		if port.Proto() != "" && k.Proto() != port.Proto() {
			continue
		}
		if len(p) == 0 {
			continue
		}
		return nat.NewPort(k.Proto(), p[0].HostPort)
	}

	return "", errors.New("port not found")
}

// Ports gets the exposed ports for the container.
func (c *DockerContainer) Ports(ctx context.Context) (nat.PortMap, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return nil, err
	}
	return inspect.NetworkSettings.Ports, nil
}

func (c *DockerContainer) inspectContainer(ctx context.Context) (*types.ContainerJSON, error) {
	inspect, err := c.provider.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	return &inspect, nil
}

// Logs will fetch both STDOUT and STDERR from the current container. Returns a
// ReadCloser and leaves it up to the caller to extract what it wants.
func (c *DockerContainer) Logs(ctx context.Context) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	return c.provider.client.ContainerLogs(ctx, c.ID, options)
}

// FollowOutput adds a LogConsumer to be sent logs from the container's
// STDOUT and STDERR
func (c *DockerContainer) FollowOutput(consumer LogConsumer) {
	if c.consumers == nil {
		c.consumers = []LogConsumer{
			consumer,
		}
	} else {
		c.consumers = append(c.consumers, consumer)
	}
}

// Name gets the name of the container.
func (c *DockerContainer) Name(ctx context.Context) (string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return "", err
	}
	return inspect.Name, nil
}

// Networks gets the names of the networks the container is attached to.
func (c *DockerContainer) Networks(ctx context.Context) ([]string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return []string{}, err
	}

	networks := inspect.NetworkSettings.Networks

	n := []string{}

	for k := range networks {
		n = append(n, k)
	}

	return n, nil
}

// ContainerIP gets the IP address of the primary network within the container.
func (c *DockerContainer) ContainerIP(ctx context.Context) (string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return "", err
	}

	return inspect.NetworkSettings.IPAddress, nil
}

// NetworkAliases gets the aliases of the container for the networks it is attached to.
func (c *DockerContainer) NetworkAliases(ctx context.Context) (map[string][]string, error) {
	inspect, err := c.inspectContainer(ctx)
	if err != nil {
		return map[string][]string{}, err
	}

	networks := inspect.NetworkSettings.Networks

	a := map[string][]string{}

	for k := range networks {
		a[k] = networks[k].Aliases
	}

	return a, nil
}

func (c *DockerContainer) Exec(ctx context.Context, cmd []string) (int, error) {
	cli := c.provider.client
	response, err := cli.ContainerExecCreate(ctx, c.ID, types.ExecConfig{
		Cmd:    cmd,
		Detach: false,
	})
	if err != nil {
		return 0, err
	}

	err = cli.ContainerExecStart(ctx, response.ID, types.ExecStartCheck{
		Detach: false,
	})
	if err != nil {
		return 0, err
	}

	var exitCode int
	for {
		execResp, err := cli.ContainerExecInspect(ctx, response.ID)
		if err != nil {
			return 0, err
		}

		if !execResp.Running {
			exitCode = execResp.ExitCode
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return exitCode, nil
}

func (c *DockerContainer) CopyFileToContainer(ctx context.Context, hostFilePath string, containerFilePath string, fileMode int64) error {
	fileContent, err := ioutil.ReadFile(hostFilePath)
	if err != nil {
		return err
	}

	buffer := &bytes.Buffer{}

	tw := tar.NewWriter(buffer)
	defer tw.Close()

	hdr := &tar.Header{
		Name: filepath.Base(containerFilePath),
		Mode: fileMode,
		Size: int64(len(fileContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(fileContent); err != nil {
		return err
	}

	return c.provider.client.CopyToContainer(ctx, c.ID, filepath.Dir(containerFilePath), buffer, types.CopyToContainerOptions{})
}

// DockerNetwork represents a network started using Docker
type DockerNetwork struct {
	ID                string // Network ID from Docker
	Driver            string
	Name              string
	provider          *DockerProvider
	terminationSignal chan bool
}

// Remove is used to remove the network. It is usually triggered by as defer function.
func (n *DockerNetwork) Remove(ctx context.Context) error {
	select {
	// close reaper if it was created
	case n.terminationSignal <- true:
	default:
	}
	return n.provider.client.NetworkRemove(ctx, n.ID)
}

// DockerProvider implements the ContainerProvider interface
type DockerProvider struct {
	client         *client.Client
	hostCache      string
	defaultNetwork string // default container network
}

// NewDockerProvider creates a Docker provider with the EnvClient
func NewDockerProvider() (*DockerProvider, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	client.NegotiateAPIVersion(context.Background())
	p := &DockerProvider{
		client: client,
	}
	return p, nil
}

// daemonHost gets the host or ip of the Docker daemon where ports are exposed on
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
// You can use the "TC_HOST" env variable to set this yourself
func (p *DockerProvider) daemonHost(ctx context.Context) (string, error) {
	if p.hostCache != "" {
		return p.hostCache, nil
	}

	host, exists := os.LookupEnv("TC_HOST")
	if exists {
		p.hostCache = host
		return p.hostCache, nil
	}

	// infer from Docker host
	url, err := url.Parse(p.client.DaemonHost())
	if err != nil {
		return "", err
	}

	switch url.Scheme {
	case "http", "https", "tcp":
		p.hostCache = url.Hostname()
	case "unix", "npipe":
		if inAContainer() {
			ip, err := p.GetGatewayIP(ctx)
			if err != nil {
				// fallback to getDefaultGatewayIP
				ip, err = getDefaultGatewayIP()
				if err != nil {
					ip = "localhost"
				}
			}
			p.hostCache = ip
		} else {
			p.hostCache = "localhost"
		}
	default:
		return "", errors.New("Could not determine host through env or docker host")
	}

	return p.hostCache, nil
}

// GetNetwork returns the object representing the network identified by its name
func (p *DockerProvider) GetNetwork(ctx context.Context, req NetworkRequest) (types.NetworkResource, error) {
	networkResource, err := p.client.NetworkInspect(ctx, req.Name, types.NetworkInspectOptions{
		Verbose: true,
	})
	if err != nil {
		return types.NetworkResource{}, err
	}

	return networkResource, err
}

func (p *DockerProvider) GetGatewayIP(ctx context.Context) (string, error) {
	// Use a default network as defined in the DockerProvider
	var err error
	if p.defaultNetwork == "" {
		p.defaultNetwork, err = getDefaultNetwork(ctx, p.client)
		if err != nil {
			return "", err
		}
	}
	nw, err := p.GetNetwork(ctx, NetworkRequest{Name: p.defaultNetwork})
	if err != nil {
		return "", err
	}

	var ip string
	for _, config := range nw.IPAM.Config {
		if config.Gateway != "" {
			ip = config.Gateway
			break
		}
	}
	if ip == "" {
		return "", errors.New("Failed to get gateway IP from network settings")
	}

	return ip, nil
}

func inAContainer() bool {
	// see https://github.com/testcontainers/testcontainers-java/blob/3ad8d80e2484864e554744a4800a81f6b7982168/core/src/main/java/org/testcontainers/dockerclient/DockerClientConfigUtils.java#L15
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// deprecated
// see https://github.com/testcontainers/testcontainers-java/blob/master/core/src/main/java/org/testcontainers/dockerclient/DockerClientConfigUtils.java#L46
func getDefaultGatewayIP() (string, error) {
	// see https://github.com/testcontainers/testcontainers-java/blob/3ad8d80e2484864e554744a4800a81f6b7982168/core/src/main/java/org/testcontainers/dockerclient/DockerClientConfigUtils.java#L27
	cmd := exec.Command("sh", "-c", "ip route|awk '/default/ { print $3 }'")
	stdout, err := cmd.Output()
	if err != nil {
		return "", errors.New("Failed to detect docker host")
	}
	ip := strings.TrimSpace(string(stdout))
	if len(ip) == 0 {
		return "", errors.New("Failed to parse default gateway IP")
	}
	return string(ip), nil
}

func getDefaultNetwork(ctx context.Context, cli *client.Client) (string, error) {
	// Get list of available networks
	networkResources, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return "", err
	}

	reaperNetwork := ReaperDefault

	reaperNetworkExists := false

	for _, net := range networkResources {
		if net.Name == Bridge {
			return Bridge, nil
		}

		if net.Name == reaperNetwork {
			reaperNetworkExists = true
		}
	}

	// Create a bridge network for the container communications
	if !reaperNetworkExists {
		_, err = cli.NetworkCreate(ctx, reaperNetwork, types.NetworkCreate{
			Driver:     Bridge,
			Attachable: true,
			Labels: map[string]string{
				TestcontainerLabel: "true",
			},
		})

		if err != nil {
			return "", err
		}
	}

	return reaperNetwork, nil
}

// WaitUntilReady implements Strategy.WaitUntilReady
func WaitPort(ctx context.Context, target wait.StrategyTarget, waitPort nat.Port, timeout time.Duration) (err error) {
	// limit context to startupTimeout
	ctx, cancelContext := context.WithTimeout(ctx, timeout)
	defer cancelContext()

	ipAddress, err := target.Host(ctx)
	if err != nil {
		return
	}

	var waitInterval = 100 * time.Millisecond

	var port nat.Port
	port, err = target.MappedPort(ctx, waitPort)
	var i = 0

	for port == "" {
		i++

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s:%w", ctx.Err(), err)
		case <-time.After(waitInterval):
			port, err = target.MappedPort(ctx, waitPort)
			if err != nil {
				fmt.Printf("(%d) [%s] %s\n", i, port, err)
			}
		}
	}

	proto := port.Proto()
	portNumber := port.Int()
	portString := strconv.Itoa(portNumber)

	//external check
	dialer := net.Dialer{}
	address := net.JoinHostPort(ipAddress, portString)
	for {
		conn, err := dialer.DialContext(ctx, proto, address)
		if err != nil {
			if v, ok := err.(*net.OpError); ok {
				if v2, ok := (v.Err).(*os.SyscallError); ok {
					if isConnRefusedErr(v2.Err) {
						time.Sleep(waitInterval)
						continue
					}
				}
			}
			return err
		} else {
			conn.Close()
			break
		}
	}

	//internal check
	command := buildInternalCheckCommand(waitPort.Int())
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		exitCode, err := target.Exec(ctx, []string{"/bin/sh", "-c", command})
		if err != nil {
			return err
		}

		if exitCode == 0 {
			break
		} else if exitCode == 126 {
			return errors.New("/bin/sh command not executable")
		}
	}

	return nil
}

func isConnRefusedErr(err error) bool {
	return err == syscall.ECONNREFUSED
}

func buildInternalCheckCommand(internalPort int) string {
	command := `(
					cat /proc/net/tcp* | awk '{print $2}' | grep -i :%04x ||
					nc -vz -w 1 localhost %d ||
					/bin/sh -c '</dev/tcp/localhost/%d'
				)
				`
	return "true && " + fmt.Sprintf(command, internalPort, internalPort, internalPort)
}
