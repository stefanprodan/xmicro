package xconsul

import (
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
