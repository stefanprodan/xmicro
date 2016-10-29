package xproxy

import (
	"errors"
	"fmt"
	"sync"

	consul "github.com/hashicorp/consul/api"
)

// Registry in memory map of elected leaders and services
type Registry struct {
	Catalog map[string][]string
	lock    sync.RWMutex
}

// Lookup returns service endpoints
func (reg Registry) Lookup(service string) ([]string, error) {
	reg.lock.RLock()
	targets, ok := reg.Catalog[service]
	reg.lock.RUnlock()
	if !ok {
		return nil, errors.New("service " + service + " not found")
	}
	return targets, nil
}

// GetServices gets elected leaders snd services from Consul
func (reg Registry) GetServices(electionKeyPrefix string) error {

	registry := make(map[string][]string)

	config := consul.DefaultConfig()
	c, err := consul.NewClient(config)
	if err != nil {
		return err
	}

	services, _, err := c.Catalog().Services(nil)
	if err != nil {
		return err
	}
	for service := range services {
		// TODO: get only healthy services (the 15s health check startup delay could be a problem)
		services, _, err := c.Health().Service(service, "", false, nil)
		if err != nil {
			return err
		}

		for _, s := range services {
			if s.Service.Address == "" {
				continue
			}
			// detect if service is subject to leader election
			if len(s.Service.Tags) >= 2 && s.Service.Tags[0] == "le" {
				// compose election key using the second tag
				var electionKey = electionKeyPrefix + s.Service.Tags[1]
				kvpair, _, err := c.KV().Get(electionKey, nil)
				if kvpair != nil && err == nil {
					// check if a session is locking the key
					sessionInfo, _, err := c.Session().Info(kvpair.Session, nil)
					if err == nil && sessionInfo != nil {
						// extract leader name from session name and validate
						_, present := registry[s.Service.Tags[1]]
						if !present && service == sessionInfo.Name {
							// add service to registry using the tag only if the current service is the leader
							registry[s.Service.Tags[1]] = append(registry[s.Service.Tags[1]], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
						}
					} else {
						return err
					}
				}
			} else {
				// add service for load balancing
				registry[service] = append(registry[service], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
			}
		}
	}

	// update local registry
	reg.lock.Lock()
	defer reg.lock.Unlock()
	for k := range reg.Catalog {
		delete(reg.Catalog, k)
	}
	for k, v := range registry {
		reg.Catalog[k] = v
	}

	return nil
}
