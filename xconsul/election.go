package xconsul

import (
	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

//Election holds the Consul leader election lock, config and status
type Election struct {
	electionKey string
	isLeader    bool
	consulLock  *consul.Lock
	stopChan    chan struct{}
	lockChan    chan struct{}
}

func (e *Election) start() {
	stop := false
	for !stop {
		select {
		case <-e.stopChan:
			stop = true
		default:
			leader := e.GetLeader()
			if leader != "" {
				log.Info("Leader is %s", leader)
			} else {
				log.Info("No leader found, starting election...")
			}
			electionChan, err := e.consulLock.Lock(e.lockChan)
			if err != nil {
				log.Warnf("Failed to acquire election lock %s", err.Error())
			}
			if electionChan != nil {
				log.Info("Acting as elected leader.")
				e.isLeader = true
				<-electionChan
				e.isLeader = false
				log.Warn("Leadership lost, releasing lock.")
				e.consulLock.Unlock()
			} else {
				log.Info("Retrying election in 5s")
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
func BeginElection(serviceName string, keyPrefix string, role string) *Election {
	key := keyPrefix + role
	config := consul.DefaultConfig()
	client, _ := consul.NewClient(config)
	opts := &consul.LockOptions{
		Key: key,
		SessionOpts: &consul.SessionEntry{
			Name:      serviceName,
			LockDelay: time.Duration(5 * time.Second),
			TTL:       "10s",
		},
	}
	lock, _ := client.LockOpts(opts)
	election := &Election{
		electionKey: key,
		consulLock:  lock,
		stopChan:    make(chan struct{}, 1),
		lockChan:    make(chan struct{}, 1),
	}
	go election.start()
	return election
}

//GetLeader returns leader name from Consul session
func (e *Election) GetLeader() string {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return ""
	}
	kvpair, _, err := client.KV().Get(e.electionKey, nil)
	if kvpair != nil && err == nil {
		sessionInfo, _, err := client.Session().Info(kvpair.Session, nil)
		if err == nil {
			return sessionInfo.Name
		}
	}
	return ""
}

//IsLeader returns true if the current instance is acting as leader
func (e *Election) IsLeader() bool {
	return e.isLeader
}
