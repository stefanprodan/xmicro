package xconsul

import (
	"log"
	"time"

	consul "github.com/hashicorp/consul/api"
)

var electionKey = ""

type Election struct {
	isLeader   bool
	consulLock *consul.Lock
	stopChan   chan struct{}
	lockChan   chan struct{}
}

func (e *Election) start() {
	stop := false
	for !stop {
		select {
		case <-e.stopChan:
			stop = true
		default:
			leader := GetLeader()
			if leader != "" {
				log.Printf("Leader is %s", leader)
			} else {
				log.Printf("No leader found, starting election...")
			}
			electionChan, err := e.consulLock.Lock(e.lockChan)
			if err != nil {
				log.Printf("Failed to acquire election lock %s", err.Error())
			}
			if electionChan != nil {
				log.Printf("Acting as elected leader.")
				e.isLeader = true
				<-electionChan
				e.isLeader = false
				log.Println("Leadership lost, releasing lock.")
				e.consulLock.Unlock()
			} else {
				log.Println("Retrying election in 5s")
				time.Sleep(5000 * time.Millisecond)
			}
		}
	}
}

//Stop ends the election routine and releases the lock
func (e *Election) Stop() {
	e.stopChan <- struct{}{}
	e.lockChan <- struct{}{}
	e.consulLock.Unlock()
	e.isLeader = false
}

//BeginElection starts a leader election on a go routine
func BeginElection(serviceName string, key string) *Election {
	electionKey = key
	config := consul.DefaultConfig()
	client, _ := consul.NewClient(config)
	opts := &consul.LockOptions{
		Key: electionKey,
		SessionOpts: &consul.SessionEntry{
			Name:      serviceName,
			LockDelay: time.Duration(5 * time.Second),
			TTL:       "10s",
		},
	}
	lock, _ := client.LockOpts(opts)
	election := &Election{
		consulLock: lock,
		stopChan:   make(chan struct{}, 1),
		lockChan:   make(chan struct{}, 1),
	}
	go election.start()
	return election
}

//GetLeader returns leader name from Consul session
func GetLeader() string {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return ""
	}
	kvpair, _, err := client.KV().Get(electionKey, nil)
	if kvpair != nil && err == nil {
		sessionInfo, _, err := client.Session().Info(kvpair.Session, nil)
		if err == nil {
			return sessionInfo.Name
		}
	}
	return ""
}

//IsLeaderElected returns true if a leader has been elected
func IsLeaderElected() bool {
	return GetLeader() != ""
}

//Leader returns leader's name
func (e *Election) Leader() string {
	return GetLeader()
}

//IsLeader returns true if the current instance is acting as leader
func (e *Election) IsLeader() bool {
	return e.isLeader
}
