package betterweb

import (
	"btrzaws"
	"fmt"
	"log"
	"logging"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

type InstancesChecker struct {
	faultyInstances             map[string]int
	restartedServicesCounterMap map[string]restartCounter
	restartingInstances         map[string]restartCounter
	lastOKLogLine               time.Time
	clientResponse              *ClientResponse
	sess                        *session.Session
	Configurations              InstancesCheckerConfiguration
	tempCheckedInstances        []*btrzaws.BetterezInstance
}

type InstancesCheckerConfiguration struct {
	Environment          string
	NotificationsOptions []string
}

func (ic *InstancesChecker) initChecker(sess *session.Session) {
	ic.sess = sess
	ic.faultyInstances = make(map[string]int)
	ic.restartedServicesCounterMap = make(map[string]restartCounter)
	ic.restartingInstances = make(map[string]restartCounter)
	ic.lastOKLogLine = time.Now().Add(ServerAliveDurationNotification)
	ic.clientResponse = &ClientResponse{Version: "1.0.0.4"}
	ic.Configurations.Environment = os.Getenv("env")
	if ic.Configurations.Environment == "" {
		ic.Configurations.Environment = "production"
	}
}

func (ic *InstancesChecker) CheckInstances(sess *session.Session) {
	ic.initChecker(sess)
	go func() {
		updateTime := time.Now()
		for {
			err := ic.getInstances()
			if err != nil {
				log.Fatalln(err, "getting instances")
			}
			ic.scanInstances()
			for !time.Now().After(updateTime.Add(time.Second * 9)) {
				time.Sleep(time.Second)
			}
			updateTime = time.Now()
			ic.updateResponseInstances()
		}
	}()
}

func (ic *InstancesChecker) updateResponseInstances() {
	ic.clientResponse.Instances = ic.clientResponse.Instances[:0]
	for _, instance := range ic.tempCheckedInstances {
		ic.clientResponse.Instances = append(ic.clientResponse.Instances, instance)
	}
}

func (ic *InstancesChecker) getTags() []*btrzaws.AwsTag {
	instanceTag := &btrzaws.AwsTag{TagName: "tag:Nginx-Configuration", TagValues: []string{"api", "app", "connex"}}
	tags := []*btrzaws.AwsTag{
		btrzaws.NewWithValues("tag:Environment", ic.Configurations.Environment),
		btrzaws.NewWithValues("tag:Service-Type", "http"),
		btrzaws.NewWithValues("tag:Online", "yes"),
		btrzaws.NewWithValues("instance-state-name", "running"),
		instanceTag,
	}
	return tags
}

func (ic *InstancesChecker) getInstances() error {
	reservations, err := btrzaws.GetInstancesWithTags(ic.sess, ic.getTags())
	if err != nil {
		return err
	}
	ic.tempCheckedInstances = ic.tempCheckedInstances[:0]
	for idx := range reservations {
		for _, instance := range reservations[idx].Instances {
			ic.tempCheckedInstances = append(ic.tempCheckedInstances, btrzaws.LoadFromAWSInstance(instance))
		}
	}
	return err
}

func (ic *InstancesChecker) instanceShouldSkipChecking(instance *btrzaws.BetterezInstance) bool {
	if isThisInstanceStillStarting(instance.InstanceID, &ic.restartingInstances) {
		logging.RecordLogLine(fmt.Sprintf("  instanceId = %s  checked = false  reason = restarting  ", instance.InstanceID))
		return true
	}
	if isThisInstanceJustCreated(instance) {
		logging.RecordLogLine(fmt.Sprintf("  instanceId = %s  checked = false  reason = new  ", instance.InstanceID))
		return true
	}
	return false
}

func (ic *InstancesChecker) scanInstances() {
	instancesIndex := 0
	for _, instance := range ic.tempCheckedInstances {
		instancesIndex++
		if ic.instanceShouldSkipChecking(instance) {
			continue
		}
		instanceIsFaulty := false
		ok, err := instance.CheckInstanceHealth()
		if err != nil {
			logging.RecordLogLine(fmt.Sprintf("warning: error %v while checking instance! Fault counted.", err))
			instanceIsFaulty = true
		} else {
			if ok {
				ic.handleWorkingInstance(instance)
			}
		}
		if instanceIsFaulty {
			ic.handleFaultyInstance(instance)
		}
	}
}

func (ic *InstancesChecker) wasInstanceFaulty(instance *btrzaws.BetterezInstance) bool {
	return ic.faultyInstances[instance.InstanceID] > 0
}

func (ic *InstancesChecker) resetRestartCounterForInstance(instance *btrzaws.BetterezInstance) {
	ic.restartingInstances[instance.InstanceID] = restartCounter{
		countingPoint:     0,
		restartCheckpoint: time.Now(),
	}
}

func (ic *InstancesChecker) setInstanceAsHealthy(instance *btrzaws.BetterezInstance) {
	ic.faultyInstances[instance.InstanceID] = 0
}

func (ic *InstancesChecker) handleWorkingInstance(instance *btrzaws.BetterezInstance) {
	if ic.wasInstanceFaulty(instance) {
		logging.RecordLogLine(fmt.Sprintf("info: Service %s on %s is back to normal.", instance.Repository, instance.InstanceID))
	}
	if ic.restartedServicesCounterMap[instance.InstanceID].countingPoint > 0 &&
		ic.restartedServicesCounterMap[instance.InstanceID].restartCheckpoint.Before(time.Now()) {
		logging.RecordLogLine(fmt.Sprintf("info: Clearing Service %s on %s notification counter.", instance.Repository, instance.InstanceID))
		ic.resetRestartCounterForInstance(instance)
	}
	ic.setInstanceAsHealthy(instance)
}

func (ic *InstancesChecker) increaseInstanceFaultCount(instance *btrzaws.BetterezInstance) {
	ic.faultyInstances[instance.InstanceID] = ic.faultyInstances[instance.InstanceID] + 1
}

func (ic *InstancesChecker) recordFailureWarning(instance *btrzaws.BetterezInstance) {
	logging.RecordLogLine(fmt.Sprintf("warning: Instance %s (%s) failed healthcheck, %d failure count.",
		instance.InstanceID, instance.Repository,
		ic.faultyInstances[instance.InstanceID]))
}

func (ic *InstancesChecker) increaseInstanceRestartCounter(instance *btrzaws.BetterezInstance) {
	ic.restartedServicesCounterMap[instance.InstanceID] = restartCounter{
		countingPoint:     ic.restartedServicesCounterMap[instance.InstanceID].countingPoint + 1,
		restartCheckpoint: time.Now().Add(time.Hour * 1),
	}
}

func (ic *InstancesChecker) setInstanceRestartCounter(instance *btrzaws.BetterezInstance) {
	ic.restartingInstances[instance.InstanceID] = restartCounter{
		countingPoint:     1,
		restartCheckpoint: time.Now().Add(HardRestartDuration),
	}
}

func (ic *InstancesChecker) restartInstance(instance *btrzaws.BetterezInstance) {
	logging.RecordLogLine(fmt.Sprintf("fatal: server %s (%s) is out, restarting", instance.InstanceID, instance.Repository))
	err := instance.RestartService()
	if err != nil {
		if instance.ShouldTerminateOnFault() {
			logging.RecordLogLine("Server %s is marked for termination. Terminating")
			instance.TerminateInstance()
			return
		}
		logging.RecordLogLine(fmt.Sprintf("fatal: error %v while restarting the service on %s (%s). Performing full restart!",
			err, instance.InstanceID, instance.Repository))
		instance.RestartServer()
		ic.setInstanceRestartCounter(instance)
	} else {
		logging.RecordLogLine(fmt.Sprintf("info: service %s (on %s) restarted.",
			instance.Repository,
			instance.InstanceID))
		ic.restartingInstances[instance.InstanceID] = restartCounter{
			countingPoint:     1,
			restartCheckpoint: time.Now().Add(SoftRestartDuration),
		}
	}
}

func (ic *InstancesChecker) handleFaultyInstance(instance *btrzaws.BetterezInstance) {
	ic.increaseInstanceFaultCount(instance)
	ic.recordFailureWarning(instance)
	if ic.faultyInstances[instance.InstanceID] > RestartThreshold {
		logging.RecordLogLine(fmt.Sprintf("info: %d restarts out of %d before notifying",
			ic.restartedServicesCounterMap[instance.InstanceID].countingPoint, ReportingThreshold))
		if ic.restartedServicesCounterMap[instance.InstanceID].countingPoint >= ReportingThreshold {
			if instance.IsInstanceOnAutoScalingGroup() {
				logging.RecordLogLine(fmt.Sprintf("Terminating %s. it's on a scaling group. no notification will be sent", instance.InstanceID))
				instance.TerminateInstance()
			} else {
				notifyInstaneFailureStatus(instance, ic.sess)
			}
		}
		ic.increaseInstanceRestartCounter(instance)
		ic.restartInstance(instance)
	}
}
