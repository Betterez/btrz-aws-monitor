package betterweb

import (
	"btrzaws"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"logging"
	"time"
)

const (
	// ReportingThreshold - how many failed attempts before notification
	ReportingThreshold = 3
	// RestartThreshold - how many failed attempts befor reset
	RestartThreshold = 3
	// TestDuration - time to wait between testing
	TestDuration = 8 * time.Second
	// SoftRestartDuraion - Time to wait till a service restarted
	SoftRestartDuraion = time.Second * 30
	// HardRestartDuration - Time to wait after a hard restart was scheduled
	HardRestartDuration = time.Second * 180
	// NotificationResetDuration - time to reset notification for service restarts
	NotificationResetDuration = time.Hour * 1
)

type restartCounter struct {
	restartCheckpoint time.Time
	countingPoint     int
}

func checkInstances(sess *session.Session, clientResponse *ClientResponse) {
	faultyInstances := make(map[string]int)
	restartedServicesCounterMap := make(map[string]restartCounter)
	restartingInstances := make(map[string]restartCounter)
	for {
		instanceTag := &btrzaws.AwsTag{TagName: "tag:Nginx-Configuration", TagValues: []string{"api", "app", "connex"}}
		tags := []*btrzaws.AwsTag{
			btrzaws.NewWithValues("tag:Environment", "production"),
			btrzaws.NewWithValues("tag:Service-Type", "http"),
			btrzaws.NewWithValues("tag:Online", "yes"),
			btrzaws.NewWithValues("instance-state-name", "running"),
			instanceTag,
		}
		for {
			reservations, err := btrzaws.GetInstancesWithTags(sess, tags)
			if err != nil {
				log.Println("Error", err, " pulling instances data")
			} else {
				clientResponse.Instances = clientResponse.Instances[:0]
				for idx := range reservations {
					for _, instance := range reservations[idx].Instances {
						clientResponse.Instances = append(clientResponse.Instances, btrzaws.LoadFromAWSInstance(instance))
					}
				}
			}
			instancesIndex := 0
			for _, instance := range clientResponse.Instances {
				instancesIndex++
				if restartingInstances[instance.InstanceID].countingPoint != 0 {
					if time.Now().Before(restartingInstances[instance.InstanceID].restartCheckpoint) {
						continue
					}
				}
				isThisInstanceFaulty := false
				ok, err := instance.CheckIsnstanceHealth()
				if err != nil {
					logging.RecordLogLine(fmt.Sprintf("error %v while checking instance! Fault counted.", err))
					isThisInstanceFaulty = true
				} else {
					if ok {
						if faultyInstances[instance.InstanceID] > 0 {
							logging.RecordLogLine(fmt.Sprintf("Service %s on %s is back to normal.", instance.Repository, instance.InstanceID))
						}
						if restartedServicesCounterMap[instance.InstanceID].countingPoint > 0 &&
							restartedServicesCounterMap[instance.InstanceID].restartCheckpoint.Before(time.Now()) {
							logging.RecordLogLine(fmt.Sprintf("Clearing Service %s on %s notification counter.", instance.Repository, instance.InstanceID))
							restartedServicesCounterMap[instance.InstanceID] = restartCounter{
								countingPoint:     0,
								restartCheckpoint: time.Now(),
							}
						}
						faultyInstances[instance.InstanceID] = 0
						restartingInstances[instance.InstanceID] = restartCounter{
							countingPoint:     0,
							restartCheckpoint: time.Now(),
						}
					} else {
						isThisInstanceFaulty = true
					}
					// NOTE: what is this? I don't think it works.
					//	instance.FaultsCount = faultyInstances[instance.InstanceID]
				}
				if isThisInstanceFaulty {
					faultyInstances[instance.InstanceID] = faultyInstances[instance.InstanceID] + 1
					logging.RecordLogLine(fmt.Sprintf("Instance %s (%s) failed healthcheck, %d failure count.",
						instance.InstanceID, instance.Repository,
						faultyInstances[instance.InstanceID]))
					// TODO: should this be =>?
					if faultyInstances[instance.InstanceID] > RestartThreshold {
						logging.RecordLogLine(fmt.Sprintf("%d restarts out of %d before notifying", restartedServicesCounterMap[instance.InstanceID].countingPoint, ReportingThreshold))
						if restartedServicesCounterMap[instance.InstanceID].countingPoint >= ReportingThreshold {
							notifyInstaneFailureStatus(instance, sess)
						}
						restartedServicesCounterMap[instance.InstanceID] = restartCounter{
							countingPoint:     restartedServicesCounterMap[instance.InstanceID].countingPoint + 1,
							restartCheckpoint: time.Now().Add(time.Hour * 1),
						}
						logging.RecordLogLine(fmt.Sprintf("server %s (%s) is out, restarting", instance.InstanceID, instance.Repository))
						err = instance.RestartService()
						if err != nil {
							logging.RecordLogLine(fmt.Sprintf("error %v while restarting the service on %s (%s). Performing full restart!",
								err, instance.InstanceID, instance.Repository))
							instance.RestartServer()
							restartingInstances[instance.InstanceID] = restartCounter{
								countingPoint:     1,
								restartCheckpoint: time.Now().Add(HardRestartDuration),
							}
						} else {
							logging.RecordLogLine(fmt.Sprintf("service %s (on %s) restarted.",
								instance.Repository,
								instance.InstanceID))
							restartingInstances[instance.InstanceID] = restartCounter{
								countingPoint:     1,
								restartCheckpoint: time.Now().Add(SoftRestartDuraion),
							}
						}
					}
				}
			}
			clientResponse.TimeStamp = time.Now()
			time.Sleep(TestDuration)
		}
	}
}

func notifyInstaneFailureStatus(faultyInstance *btrzaws.BetterezInstance, sess *session.Session) {
	logging.RecordLogLine(fmt.Sprintf("instance %s failure notice was sent. repo: %s", faultyInstance.InstanceID, faultyInstance.Repository))
	btrzaws.Notify(faultyInstance, sess)
}
