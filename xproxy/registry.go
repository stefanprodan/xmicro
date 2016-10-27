package xproxy

import (
	"errors"
	"sync"
	//"log"

	//consul "github.com/hashicorp/consul/api"
	//watch "github.com/hashicorp/consul/watch"
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
