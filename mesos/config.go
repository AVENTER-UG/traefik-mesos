package mesos

import (
	"context"

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
			labels[label.Key] = label.Value
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
		provider.BuildRouterConfiguration(ctx, confFromLabel.HTTP, task.ID, p.defaultRuleTpl, model)

		configurations[containerName] = confFromLabel
	}

	return provider.Merge(ctx, configurations)
}
