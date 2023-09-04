package mesos

import (
	"context"
	"net"
	"strconv"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

// buildUDPServiceConfiguration buid the UDP Service of the Mesos Taks
// containerName.
func (p *Provider) buildUDPServiceConfiguration(ctx context.Context, containerName string, configuration *dynamic.UDPConfiguration) {
	if len(configuration.Routers) == 0 {
		return
	}

	var lb *dynamic.UDPServersLoadBalancer
	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.UDPService)
		lb = new(dynamic.UDPServersLoadBalancer)
	}
	// if there is no service configures, create one
	for _, service := range configuration.Services {
		if service.LoadBalancer == nil {
			lb = new(dynamic.UDPServersLoadBalancer)
		} else {
			lb = service.LoadBalancer
		}
	}

	for _, service := range configuration.Routers {
		// search all different ports by name and create a Loadbalancer configuration for traefik
		task := p.mesosConfig[containerName].Tasks[0]
		if len(task.Discovery.Ports.Ports) > 0 {
			for _, port := range task.Discovery.Ports.Ports {
				if len(port.Name) == 0 || port.Protocol != "udp" {
					continue
				}
				if port.Name != service.Service {
					continue
				}

				if len(lb.Servers) == 0 {
					server := dynamic.UDPServer{}

					lb.Servers = []dynamic.UDPServer{server}
				}

				lb.Servers = p.getUDPServers(port.Name, containerName)

				lbService := &dynamic.UDPService{
					LoadBalancer: lb,
				}

				configuration.Services[service.Service] = lbService
			}
		}
	}
}

// getUDPServers search all IP addresses to the given portName of
// the Mesos Task with the containerName.
func (p *Provider) getUDPServers(portName string, containerName string) []dynamic.UDPServer {
	var servers []dynamic.UDPServer
	for _, task := range p.mesosConfig[containerName].Tasks {
		for _, status := range task.Statuses {
			// the host ip is only visible during starting task. Have to find out why
			if status.State == "TASK_STARTING" {
				for _, network := range status.ContainerStatus.NetworkInfos {
					for _, ip := range network.IPAddresses {
						if ip.Protocol == "IPv4" && len(task.Discovery.Ports.Ports) > 0 {
							for _, port := range task.Discovery.Ports.Ports {
								if portName != port.Name || port.Protocol != "udp" {
									continue
								}
								po := strconv.Itoa(port.Number)
								server := dynamic.UDPServer{
									Address: net.JoinHostPort(ip.IPAddress, po),
									Port:    po,
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
