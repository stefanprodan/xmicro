package xconsul

import (
	"encoding/json"
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

// ConsulClient wrapper over api client
type ConsulClient struct {
	Client   *consul.Client
	Config   *consul.Config
	watchers []*watch.WatchPlan
	mutex    sync.Mutex
}

// NewClient returns a ConsulClient with defaults
func NewClient() (*ConsulClient, error) {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	c := &ConsulClient{
		Client: client,
		Config: config,
	}
	return c, nil
}

// ListServices outputs all services in Consul catalog
func (c *ConsulClient) ListServices() error {
	services, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return err
	}
	for service := range services {
		log.Printf("%v", service)

		r, _, err := c.Client.Health().Service(service, "", false, nil)
		if err != nil {
			return err
		}

		for _, s := range r {
			log.Printf("%v", s.Service)
		}
	}
	return nil
}

// GetServices returns a map of services and endpoints
func (c *ConsulClient) GetServices() (map[string][]string, error) {

	registry := make(map[string][]string)

	services, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return registry, err
	}
	for service := range services {
		r, _, err := c.Client.Health().Service(service, "", false, nil)
		if err != nil {
			return registry, err
		}

		for _, s := range r {
			registry[service] = append(registry[service], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
		}
	}
	return registry, nil
}

// GetLeaderServices returns a map of elected leaders and their endpoints
func (c *ConsulClient) GetLeaderServices(electionKeyPrefix string) (map[string][]string, error) {

	registry := make(map[string][]string)

	services, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return registry, err
	}
	for service := range services {
		r, _, err := c.Client.Health().Service(service, "", false, nil)
		if err != nil {
			return registry, err
		}

		for _, s := range r {
			//detect if service is subject to leader election
			if len(s.Service.Tags) == 2 && s.Service.Tags[0] == "le" {
				var electionKey = electionKeyPrefix + s.Service.Tags[1]
				kvpair, _, err := c.Client.KV().Get(electionKey, nil)
				if kvpair != nil && err == nil {
					//check if a session is locking the key
					sessionInfo, _, err := c.Client.Session().Info(kvpair.Session, nil)
					if err == nil && sessionInfo != nil {
						//extract leader name from session name and validate
						_, present := registry[s.Service.Tags[1]]
						if !present && service == sessionInfo.Name {
							//add service to registry using the tag as service name
							registry[s.Service.Tags[1]] = append(registry[s.Service.Tags[1]], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
						}
					} else {
						return registry, err
					}
				}
			}
		}
	}
	return registry, nil
}

// StartKeyPrefixWatcher starts a Consul watcher for the specified key prefix
func (c *ConsulClient) StartKeyPrefixWatcher(keyPrefix string, handler func(uint64, interface{})) error {
	wt, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": keyPrefix})
	if err != nil {
		return err
	}
	wt.Handler = handler
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.watchers = append(c.watchers, wt)
	go wt.Run(c.Config.Address)
	return nil
}

// LogChanges Consul watcher handler that logs changes as JSON
func (c *ConsulClient) LogChanges(idx uint64, data interface{}) {
	log.Print("Consul changes detected")
	buf, _ := json.MarshalIndent(data, "", "    ")
	log.Print(string(buf))
}

// StartServicesWatcher starts a Consul watcher for service catalog changes
func (c *ConsulClient) StartServicesWatcher(handler func(uint64, interface{})) error {
	wt, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err != nil {
		return err
	}
	wt.Handler = handler
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.watchers = append(c.watchers, wt)
	go wt.Run(c.Config.Address)
	return nil
}

// LogServicesChanges Consul catalog watcher handler that logs the registered services
func (c *ConsulClient) LogServicesChanges(idx uint64, data interface{}) {
	services, _ := data.(map[string][]string)
	log.Print("===> Catalog <===")
	for service := range services {
		log.Printf("%v", service)
		r, _, err := c.Client.Health().Service(service, "", false, nil)
		if err == nil {
			for _, s := range r {
				log.Printf("%v", s.Service)
			}
		} else {
			log.Print(err.Error())
		}
	}
	log.Print("=================")
}

// Stop stops Consul watchers
func (c *ConsulClient) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, w := range c.watchers {
		w.Stop()
	}
}
