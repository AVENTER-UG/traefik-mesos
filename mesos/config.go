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
		var task MesosTask
		// search the running task
		for _, cTask := range tasks.Tasks {
			if cTask.State == "TASK_RUNNING" {
				task = cTask
			}
		}

		if task.Labels != nil {
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
				p.logger.Warnf("Ignore Error in DecodeConfiguration (%s): %s", task.Name, err.Error())
				continue
			}

			//res2B, _ := json.Marshal(confFromLabel)
			//fmt.Println(string(res2B))

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

			//res2B, _ = json.Marshal(confFromLabel)
			//fmt.Println(string(res2B))

			configurations[containerName] = confFromLabel
		}
	}

	return provider.Merge(ctx, configurations)
}
