package xproxy

import (
	"errors"
	"fmt"
	"sync"

	consul "github.com/hashicorp/consul/api"
)

var lock sync.RWMutex

type Registry map[string][]string

func (reg Registry) Lookup(service string) ([]string, error) {
	lock.RLock()
	targets, ok := reg[service]
	lock.RUnlock()
	if !ok {
		return nil, errors.New("service " + service + " not found")
	}
	return targets, nil
}

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
	for service, _ := range services {
		r, _, err := c.Health().Service(service, "", false, nil)
		if err != nil {
			return err
		}

		for _, s := range r {
			//detect if service is subject to leader election
			if len(s.Service.Tags) == 2 && s.Service.Tags[0] == "le" {
				var electionKey = electionKeyPrefix + s.Service.Tags[1]
				kvpair, _, err := c.KV().Get(electionKey, nil)
				if kvpair != nil && err == nil {
					//check if a session is locking the key
					sessionInfo, _, err := c.Session().Info(kvpair.Session, nil)
					if err == nil && sessionInfo != nil {
						//extract leader name from session name and validate
						_, present := registry[s.Service.Tags[1]]
						if !present && service == sessionInfo.Name {
							//add service to registry using the tag only if the current service is the leader
							registry[s.Service.Tags[1]] = append(registry[s.Service.Tags[1]], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
						}
					} else {
						return err
					}
				}
			} else {
				registry[service] = append(registry[service], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
			}
		}
	}

	lock.Lock()
	for k, v := range registry {
		reg[k] = v
	}

	lock.Unlock()

	return nil
}
