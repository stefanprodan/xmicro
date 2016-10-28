package xconsul

import (
	"encoding/json"
	"fmt"
	"log"

	consul "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

func NewClient() (*consul.Client, error) {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return client, err
	}

	return client, nil
}

func ListServices(c *consul.Client) error {
	services, _, err := c.Catalog().Services(nil)
	if err != nil {
		return err
	}
	for service, _ := range services {
		log.Printf("%v", service)

		r, _, err := c.Health().Service(service, "", false, nil)
		if err != nil {
			return err
		}

		for _, s := range r {
			log.Printf("%v", s.Service)
		}
	}
	return nil
}

func GetServices(c *consul.Client) (map[string][]string, error) {

	registry := make(map[string][]string)

	services, _, err := c.Catalog().Services(nil)
	if err != nil {
		return registry, err
	}
	for service, _ := range services {
		r, _, err := c.Health().Service(service, "", false, nil)
		if err != nil {
			return registry, err
		}

		for _, s := range r {
			registry[service] = append(registry[service], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
		}
	}
	return registry, nil
}

func GetLeaderServices(c *consul.Client, electionKeyPrefix string) (map[string][]string, error) {

	registry := make(map[string][]string)

	services, _, err := c.Catalog().Services(nil)
	if err != nil {
		return registry, err
	}
	for service, _ := range services {
		r, _, err := c.Health().Service(service, "", false, nil)
		if err != nil {
			return registry, err
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

func StartElectionWatcher(keyPrefix string) error {
	wt, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": keyPrefix})
	if err != nil {
		return err
	}
	wt.Handler = handleLeaderChanges
	config := consul.DefaultConfig()
	go wt.Run(config.Address)
	return nil
}

func handleLeaderChanges(idx uint64, data interface{}) {
	log.Print("Leader change detected")
	buf, _ := json.MarshalIndent(data, "", "    ")
	log.Print(string(buf))
}

func StartServicesWatcher() error {
	wt, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err != nil {
		return err
	}
	wt.Handler = handleServicesChanges
	config := consul.DefaultConfig()
	go wt.Run(config.Address)
	return nil
}

func handleServicesChanges(idx uint64, data interface{}) {
	services, _ := data.(map[string][]string)
	log.Print("===> Registry <===")
	config := consul.DefaultConfig()
	c, _ := consul.NewClient(config)
	for service, _ := range services {
		log.Printf("%v", service)
		r, _, err := c.Health().Service(service, "", false, nil)
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
