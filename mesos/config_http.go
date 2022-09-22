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
	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.Service)
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
				lb := &dynamic.ServersLoadBalancer{}
				lb.SetDefaults()
				lb.Servers = p.getHTTPServers(port.Name, containerName)

				lbService := &dynamic.Service{
					LoadBalancer: lb,
				}
				//				res2B, _ := json.Marshal(lbService)
				//				fmt.Println(string(res2B))

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
								if portName != port.Name || port.Protocol != "tcp" {
									continue
								}
								po := strconv.Itoa(port.Number)
								server := dynamic.Server{
									URL: fmt.Sprintf("http://%s", net.JoinHostPort(ip.IPAddress, po)),
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
