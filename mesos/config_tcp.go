package mesos

import (
	"context"
	"net"
	"strconv"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// buildTCPServiceConfiguration buid the TCP Service of the Mesos Taks
// containerName.
func (p *Provider) buildTCPServiceConfiguration(ctx context.Context, containerName string, configuration *dynamic.TCPConfiguration) {
	if len(configuration.Routers) == 0 {
		return
	}

	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.TCPService)
	}

	for _, service := range configuration.Routers {
		// search all different ports by name and create a Loadbalancer configuration for traefik
		task := p.mesosConfig[containerName].Tasks[0]
		if len(task.Discovery.Ports.Ports) > 0 {
			for _, port := range task.Discovery.Ports.Ports {
				if len(port.Name) == 0 || port.Protocol != "tcp" {
					continue
				}
				if port.Name != service.Service {
					continue
				}

				var lb *dynamic.TCPServersLoadBalancer
				if configuration.Services[service.Service] == nil {
					configuration.Services[service.Service] = &dynamic.TCPService{}
				}
				if configuration.Services[service.Service].LoadBalancer == nil {
					lb = new(dynamic.TCPServersLoadBalancer)
				} else {
					lb = configuration.Services[service.Service].LoadBalancer
				}

				if len(lb.Servers) == 0 {
					server := dynamic.TCPServer{}
					lb.Servers = []dynamic.TCPServer{server}
				}

				if len(lb.Servers) == 0 {
					server := dynamic.TCPServer{}

					lb.Servers = []dynamic.TCPServer{server}
				}

				lb.Servers = p.getTCPServers(port.Name, containerName, lb)

				lbService := &dynamic.TCPService{
					LoadBalancer: lb,
				}

				configuration.Services[service.Service] = lbService
			}
		}
	}
}

// getTCPServers search all IP addresses to the given portName of
// the Mesos Task with the containerName.
func (p *Provider) getTCPServers(portName string, containerName string, lb *dynamic.TCPServersLoadBalancer) []dynamic.TCPServer {
	var servers []dynamic.TCPServer
	for _, task := range p.mesosConfig[containerName].Tasks {
		for _, status := range task.Statuses {
			// the host ip is only visible during starting task. Have to find out why
			if status.State == "TASK_STARTING" {
				for _, network := range status.ContainerStatus.NetworkInfos {
					for _, ip := range network.IPAddresses {
						if ip.Protocol == "IPv4" && len(task.Discovery.Ports.Ports) > 0 {
							for _, port := range task.Discovery.Ports.Ports {
								if portName != port.Name || port.Protocol != "tcp" {
									continue
								}
								po := strconv.Itoa(port.Number)

								if lb.Servers[0].Port != "" {
									po = lb.Servers[0].Port
								}

								server := dynamic.TCPServer{
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
