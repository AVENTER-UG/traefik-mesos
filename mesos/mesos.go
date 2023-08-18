package mesos

import (
	"context"
	cTls "crypto/tls"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"

	// Register mesos zoo the detector

	_ "github.com/mesos/mesos-go/api/v0/detector/zoo"
)

// DefaultTemplateRule The default template for the default rule.
const DefaultTemplateRule = "Host(`{{ normalize .Name }}`)"

var (
	_ provider.Provider = (*Provider)(nil)
)

// Provider holds configuration of the provider.
type Provider struct {
	Endpoint              string        `Description:"Mesos server endpoint. You can also specify multiple endpoint for Mesos"`
	SSL                   bool          `Description:"Enable Endpoint SSL"`
	Principal             string        `Description:"Principal to authorize agains Mesos Manager"`
	Secret                string        `Description:"Secret authorize agains Mesos Manager"`
	PollInterval          time.Duration `Description:"Polling interval for endpoint." json:"pollInt"`
	PollTimeout           time.Duration `Description:"Polling timeout for endpoint." json:"pollTime"`
	DefaultRule           string        `Description:"Default rule." json:"defaultRule,omitempty" toml:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	ForceUpdateInterval   time.Duration `Description:"Interval to force an update."`
	logger                log.Logger
	mesosConfig           map[string]*MesosTasks
	defaultRuleTpl        *template.Template
	lastConfigurationHash uint64
	lastUpdate            time.Time
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.Endpoint = "127.0.0.1:5050"
	p.SSL = false
	p.PollInterval = time.Duration(10 * time.Second)
	p.PollTimeout = time.Duration(10 * time.Second)
	p.DefaultRule = DefaultTemplateRule
	p.ForceUpdateInterval = time.Duration(10 * time.Minute)
	p.lastUpdate = time.Now()
}

// Init the provider.
func (p *Provider) Init() error {
	defaultRuleTpl, err := provider.MakeDefaultRuleTemplate(p.DefaultRule, nil)
	if err != nil {
		return fmt.Errorf("error while parsing default rule: %w", err)
	}

	p.defaultRuleTpl = defaultRuleTpl
	p.mesosConfig = make(map[string]*MesosTasks)
	return nil
}

// Provide allows the mesos provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctx := log.With(context.Background(), log.Str(log.ProviderName, "mesos"))
		p.logger = log.FromContext(ctx)

		// Add protocoll to the endpoint depends if SSL is enabled
		protocol := "http://" + p.Endpoint
		if p.SSL {
			protocol = "https://" + p.Endpoint
		}
		p.Endpoint = protocol

		p.logger.Info("Connect Mesos Provider to: ", p.Endpoint)

		operation := func() error {
			ticker := time.NewTicker(time.Duration(p.PollInterval))
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					tasks := p.getTasks()

					// build hash to find out if the config changes
					fnvHasher := fnv.New64()
					tasksString, _ := json.Marshal(&tasks)
					_, err := fnvHasher.Write(tasksString)

					if err != nil {
						p.logger.Errorf("cannot hash mesos tasks data: ", err.Error())
						continue
					}

					// check if the configuration has changed or the last update is 10 minutes ago
					timeNow := time.Now()
					timeDiff := timeNow.Sub(p.lastUpdate).Minutes()
					hash := fnvHasher.Sum64()

					if timeDiff >= p.ForceUpdateInterval.Minutes() {
						p.logger.Infof("Force Update Traefik Config: %d", timeDiff)
					} else {
						if hash == p.lastConfigurationHash {
							continue
						}
					}

					p.lastUpdate = timeNow
					p.lastConfigurationHash = hash
					p.logger.Info("Update Traefik Config")

					// collect all mesos tasks and combine the belong one.
					for _, task := range tasks.Tasks {
						if task.State == "TASK_RUNNING" {
							if task.Labels != nil {
								if p.checkTraefikLabels(task) {
									if p.checkContainer(task) {
										containerName := task.ID
										if p.mesosConfig[containerName] == nil {
											p.mesosConfig[containerName] = &MesosTasks{}
										}
										p.mesosConfig[containerName].Tasks = append(p.mesosConfig[containerName].Tasks, task)
									}
								}
							}
						}
					}

					// build the treafik configuration
					if len(p.mesosConfig) > 0 {
						configuration := p.buildConfiguration(ctx)
						if configuration != nil {
							configurationChan <- dynamic.Message{
								ProviderName:  "mesos",
								Configuration: configuration,
							}
						}
					}

					// cleanup old data
					p.mesosConfig = make(map[string]*MesosTasks)
				case <-routineCtx.Done():
					return nil
				}
			}
			return nil
		}
		notify := func(err error, time time.Duration) {
			p.logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
		}

		err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
		if err != nil {
			p.logger.Errorf("Cannot connect to Provider server: %+v", err)
		}
	})
	return nil
}

func (p *Provider) checkTraefikLabels(task MesosTask) bool {
	for _, label := range task.Labels {
		if strings.Contains(label.Key, "traefik.") {
			return true
		}
	}
	return false
}

func (p *Provider) getTasks() MesosTasks {
	client := &http.Client{}
	client.Transport = &http.Transport{
		TLSClientConfig: &cTls.Config{InsecureSkipVerify: true},
	}
	req, _ := http.NewRequest("GET", p.Endpoint+"/tasks?order=asc&limit=-1", nil)
	req.Close = true
	req.SetBasicAuth(p.Principal, p.Secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		p.logger.Error("Error during get tasks: ", err.Error())
		return MesosTasks{}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Errorf("received non-ok response code: %d", res.StatusCode)
		return MesosTasks{}
	}

	p.logger.Info("Get Data from Mesos")

	var tasks MesosTasks
	err = json.NewDecoder(res.Body).Decode(&tasks)
	if err != nil {
		p.logger.Error("Error in Data from Mesos: " + err.Error())
		return MesosTasks{}
	}
	return tasks
}

func (p *Provider) checkContainer(task MesosTask) bool {
	agentHostname, agentPort, err := p.getAgent(task.SlaveID)

	if err != nil {
		p.logger.Error("CheckContainer: Error in get AgentData from Mesos: " + err.Error())
		return false
	}

	p.logger.Debug("CheckContainer: " + task.Name + " on agent (" + task.SlaveID + ")" + agentHostname + " with task.ID " + task.ID)

	if agentHostname != "" {
		containers, _ := p.getContainersOfAgent(agentHostname, agentPort)

		for _, a := range containers {
			p.logger.Debug(task.ID + " --CONTAINER--  " + a.ExecutorID)
			if a.ExecutorID == task.ID {
				return true
			}
		}
	}

	return false
}

func (p *Provider) getAgent(slaveID string) (string, int, error) {
	client := &http.Client{}
	client.Transport = &http.Transport{
		TLSClientConfig: &cTls.Config{InsecureSkipVerify: true},
	}
	req, _ := http.NewRequest("GET", p.Endpoint+"/slaves/", nil)
	req.Close = true
	req.SetBasicAuth(p.Principal, p.Secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		p.logger.Error("Error during get agent: ", err.Error())
		return "", 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("received non-ok response code: %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	var agents MesosAgent
	if err := json.Unmarshal(data, &agents); err != nil {
		p.logger.Error("getAgent: Error in AgentData from Mesos  " + p.Endpoint + " with error: " + err.Error())
		return "", 0, err
	}

	for _, a := range agents.Slaves {
		if a.ID == slaveID {
			return a.Hostname, a.Port, nil
		}
	}

	return "", 0, nil
}

func (p *Provider) getContainersOfAgent(agentHostname string, agentPort int) (MesosAgentContainers, error) {
	// Add protocoll to the endpoint depends if SSL is enabled
	protocol := "http://"
	if p.SSL {
		protocol = "https://"
	}

	client := &http.Client{}
	client.Transport = &http.Transport{
		TLSClientConfig: &cTls.Config{InsecureSkipVerify: true},
	}
	req, _ := http.NewRequest("GET", protocol+agentHostname+":"+strconv.Itoa(agentPort)+"/containers/", nil)
	req.Close = true
	req.SetBasicAuth(p.Principal, p.Secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		p.logger.Error("Error during get container: ", err.Error())
		return MesosAgentContainers{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return MesosAgentContainers{}, fmt.Errorf("received non-ok response code: %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	var containers MesosAgentContainers
	if err := json.Unmarshal(data, &containers); err != nil {
		p.logger.Error("getContainersOfAgent: Error in ContainerAgentData from " + agentHostname + "  " + err.Error())
		return MesosAgentContainers{}, err
	}

	return containers, nil
}
