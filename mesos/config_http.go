package mesos

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

// buildHTTPServiceConfiguration buid th HTTP Service of the Mesos Taks
// containerName.
func (p *Provider) buildHTTPServiceConfiguration(ctx context.Context, containerName string, configuration *dynamic.HTTPConfiguration) {
	if len(configuration.Routers) == 0 {
		return
	}

	// if there is no service configures, create one
	var lb *dynamic.ServersLoadBalancer
	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.Service)
		lb = new(dynamic.ServersLoadBalancer)
		lb.SetDefaults()
	}

	for _, service := range configuration.Services {
		if service.LoadBalancer == nil {
			lb = new(dynamic.ServersLoadBalancer)
			lb.SetDefaults()
		} else {
			lb = service.LoadBalancer
		}
	}

	for _, service := range configuration.Routers {
		// search all different ports by name and create a Loadbalancer configuration for traefik
		task := p.mesosConfig[containerName].Tasks[0]
		if len(task.Discovery.Ports.Ports) > 0 {
			for _, port := range task.Discovery.Ports.Ports {
				if len(port.Name) == 0 || port.Protocol == "udp" {
					continue
				}
				if port.Name != service.Service {
					continue
				}

				if len(lb.Servers) == 0 {
					server := dynamic.Server{}
					server.SetDefaults()

					lb.Servers = []dynamic.Server{server}
				}

				lb.Servers = p.getHTTPServers(port.Name, containerName)

				lbService := &dynamic.Service{
					LoadBalancer: lb,
				}
				configuration.Services[service.Service] = lbService
			}
		}
	}
}

// getHTTPServers search all IP addresses to the given portName of
// the Mesos Task with the containerName.
func (p *Provider) getHTTPServers(portName string, containerName string) []dynamic.Server {
	var servers []dynamic.Server
	for _, task := range p.mesosConfig[containerName].Tasks {
		for _, status := range task.Statuses {
			// the host ip is only visible during starting task. Have to find out why
			if status.State == "TASK_STARTING" {
				for _, network := range status.ContainerStatus.NetworkInfos {
					for _, ip := range network.IPAddresses {
						if ip.Protocol == "IPv4" && len(task.Discovery.Ports.Ports) > 0 {
							for _, port := range task.Discovery.Ports.Ports {
								if portName != port.Name || port.Protocol == "udp" {
									continue
								}
								po := strconv.Itoa(port.Number)

								// set default protocol
								protocol := "http"

								if port.Protocol == "wss" {
									protocol = "wss"
								}
								if port.Protocol == "h2c" {
									protocol = "h2c"
								}
								if port.Protocol == "https" {
									protocol = "https"
								}
								server := dynamic.Server{
									URL: fmt.Sprintf("%s://%s", protocol, net.JoinHostPort(ip.IPAddress, po)),
								}
								servers = append(servers, server)
							}
						}
					}
				}
			}
		}
	}
	return servers
}
