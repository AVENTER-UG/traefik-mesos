package mesos

import (
	"context"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/label"
	"github.com/traefik/traefik/v2/pkg/provider"
)

func (p *Provider) buildConfiguration(ctx context.Context) *dynamic.Configuration {
	configurations := make(map[string]*dynamic.Configuration)
	labels := make(map[string]string)
	for _, tasks := range p.mesosConfig {
		task := tasks.Tasks[0]
		// The first Task is the leading one
		containerName := task.ID
		//	res2B, _ := json.Marshal(containerName)
		//fmt.Println(string(res2B))
		for _, label := range task.Labels {
			key := strings.ReplaceAll(label.Key, "__mesos_taskid__", strings.ReplaceAll(task.ID, ".", "_"))
			value := strings.ReplaceAll(label.Value, "__mesos_taskid__", strings.ReplaceAll(task.ID, ".", "_"))
			labels[key] = value
		}
		confFromLabel, err := label.DecodeConfiguration(labels)
		if err != nil {
			p.logger.Error("Error in DecodeConfiguration: " + err.Error())
			return nil
		}

		p.buildTCPServiceConfiguration(ctx, containerName, confFromLabel.TCP)
		provider.BuildTCPRouterConfiguration(ctx, confFromLabel.TCP)

		p.buildUDPServiceConfiguration(ctx, containerName, confFromLabel.UDP)
		provider.BuildUDPRouterConfiguration(ctx, confFromLabel.UDP)

		p.buildHTTPServiceConfiguration(ctx, containerName, confFromLabel.HTTP)

		model := struct {
			Name   string
			Labels map[string]string
		}{
			Name:   task.Name,
			Labels: labels,
		}
		provider.BuildRouterConfiguration(ctx, confFromLabel.HTTP, containerName, p.defaultRuleTpl, model)

		configurations[containerName] = confFromLabel
	}

	return provider.Merge(ctx, configurations)
}
