package betterweb

import (
	"btrzaws"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"logging"
	"os"
	"time"
)

type InstanceChecker struct {
	faultyInstances             map[string]int
	restartedServicesCounterMap map[string]restartCounter
	restartingInstances         map[string]restartCounter
	lastOKLogLine               time.Time
	clientResponse              *ClientResponse
	sess                        *session
}

func (ic *InstanceChecker) initChecker() {
	ic.faultyInstances = make(map[string]int)
	ic.restartedServicesCounterMap = make(map[string]restartCounter)
	ic.restartingInstances = make(map[string]restartCounter)
	ic.lastOKLogLine = time.Now().Add(ServerAliveDurationNotification)
	ic.clientResponse = &ClientResponse{Version: "1.0.0.4"}
}

func getTags() []*btrzaws.AwsTag {
	instanceTag := &btrzaws.AwsTag{TagName: "tag:Nginx-Configuration", TagValues: []string{"api", "app", "connex"}}
	tags := []*btrzaws.AwsTag{
		btrzaws.NewWithValues("tag:Environment", "production"),
		btrzaws.NewWithValues("tag:Service-Type", "http"),
		btrzaws.NewWithValues("tag:Online", "yes"),
		btrzaws.NewWithValues("instance-state-name", "running"),
		instanceTag,
	}
	return tags
}

func (ic *InstanceChecker) getInstances() error {
	reservations, err := btrzaws.GetInstancesWithTags(sess, tags)
	if err != nil {
		return err
	} else {
		ic.clientResponse.Instances = ic.clientResponse.Instances[:0]
		for idx := range reservations {
			for _, instance := range reservations[idx].Instances {
				ic.clientResponse.Instances = append(ic.clientResponse.Instances, btrzaws.LoadFromAWSInstance(instance))
			}
		}
	}
	return err
}

func (ic *InstanceChecker) scanInstances() {
	instancesIndex := 0
	for _, instance := range ic.clientResponse.Instances {
		instancesIndex++
		if isThisInsataceStillStarting(instance.InstanceID, &ic.restartingInstances) {
			logging.RecordLogLine(fmt.Sprintf("  instanceId = %s  checked = false  reason = restarting  ", instance.InstanceID))
			continue
		}
		if isThisInstanceJustCreated(instance) {
			logging.RecordLogLine(fmt.Sprintf("  instanceId = %s  checked = false  reason = new  ", instance.InstanceID))
			continue
		}
		isThisInstanceFaulty := false
		ok, err := instance.CheckIsnstanceHealth()
		if err != nil {
			logging.RecordLogLine(fmt.Sprintf("warning: error %v while checking instance! Fault counted.", err))
			isThisInstanceFaulty = true
		} else {
			if ok {
				ic.handleWorkingInstance(instance)
			}
		}
		if isThisInstanceFaulty {
			ic.handleFaultyInstance(instance)
		}
	}
}

func (ic *InstanceChecker) handleWorkingInstance(instance *btrzaws.BetterezInstance) {
	if ic.faultyInstances[instance.InstanceID] > 0 {
		logging.RecordLogLine(fmt.Sprintf("info: Service %s on %s is back to normal.", instance.Repository, instance.InstanceID))
	}
	if ic.restartedServicesCounterMap[instance.InstanceID].countingPoint > 0 &&
		ic.restartedServicesCounterMap[instance.InstanceID].restartCheckpoint.Before(time.Now()) {
		logging.RecordLogLine(fmt.Sprintf("info: Clearing Service %s on %s notification counter.", instance.Repository, instance.InstanceID))
		ic.restartedServicesCounterMap[instance.InstanceID] = restartCounter{
			countingPoint:     0,
			restartCheckpoint: time.Now(),
		}
	}
	ic.faultyInstances[instance.InstanceID] = 0
	ic.restartingInstances[instance.InstanceID] = restartCounter{
		countingPoint:     0,
		restartCheckpoint: time.Now(),
	}
}

func (ic *InstanceChecker) increaseInstanceFault(instance *btrzaws.BetterezInstance) {
	ic.faultyInstances[instance.InstanceID] = ic.faultyInstances[instance.InstanceID] + 1
}

func (ic *InstanceChecker) recordFailureWarning(instance *btrzaws.BetterezInstance) {
	logging.RecordLogLine(fmt.Sprintf("warning: Instance %s (%s) failed healthcheck, %d failure count.",
		instance.InstanceID, instance.Repository,
		ic.faultyInstances[instance.InstanceID]))
}

func (ic *InstanceChecker) updateInstanceRestartCounter(instance *btrzaws.BetterezInstance) {
	ic.restartedServicesCounterMap[instance.InstanceID] = restartCounter{
		countingPoint:     ic.restartedServicesCounterMap[instance.InstanceID].countingPoint + 1,
		restartCheckpoint: time.Now().Add(time.Hour * 1),
	}
}

func (ic *InstanceChecker) restartInstance(instance *btrzaws.BetterezInstance) {
	logging.RecordLogLine(fmt.Sprintf("fatal: server %s (%s) is out, restarting", instance.InstanceID, instance.Repository))
	err := instance.RestartService()
	if err != nil {
		logging.RecordLogLine(fmt.Sprintf("fatal: error %v while restarting the service on %s (%s). Performing full restart!",
			err, instance.InstanceID, instance.Repository))
		instance.RestartServer()
		ic.restartingInstances[instance.InstanceID] = restartCounter{
			countingPoint:     1,
			restartCheckpoint: time.Now().Add(HardRestartDuration),
		}
	} else {
		logging.RecordLogLine(fmt.Sprintf("info: service %s (on %s) restarted.",
			instance.Repository,
			instance.InstanceID))
		ic.restartingInstances[instance.InstanceID] = restartCounter{
			countingPoint:     1,
			restartCheckpoint: time.Now().Add(SoftRestartDuraion),
		}
	}
}

func (ic *InstanceChecker) handleFaultyInstance(instance *btrzaws.BetterezInstance) {
	ic.increaseInstanceFault(instance)
	ic.recordFailureWarning(instance)
	if ic.faultyInstances[instance.InstanceID] > RestartThreshold {
		logging.RecordLogLine(fmt.Sprintf("info: %d restarts out of %d before notifying", ic.restartedServicesCounterMap[instance.InstanceID].countingPoint, ReportingThreshold))
		if ic.restartedServicesCounterMap[instance.InstanceID].countingPoint >= ReportingThreshold {
			notifyInstaneFailureStatus(instance, ic.sess)
		}
		ic.updateInstanceRestartCounter(instance)
		ic.restartInstance(instance)
	}
}

func (ic *InstanceChecker) CheckInstances(sess *session.Session) {
	ic.sess = sess
	ic.initChecker()
	ic.getInstances()
	ic.scanInstances()
}
