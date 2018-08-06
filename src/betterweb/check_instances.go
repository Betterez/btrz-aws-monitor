package betterweb

import (
	"btrzaws"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/bsphere/le_go"
)

const (
	// ReportingThreshold - how many failed attempts before notification
	ReportingThreshold = 5
	// ReserThreshold - how many failed attempts befor reset
	ReserThreshold = 3
	// TestDuration - time to wait between testing
	TestDuration = 8 * time.Second
)

type restartCounter struct {
	restartPoint  time.Time
	countingPoint int
}

func checkInstances(sess *session.Session, clientResponse *ClientResponse) {
	faultyInstances := make(map[string]int)
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
					continue
				}
				ok, err := instance.CheckIsnstanceHealth()
				if err != nil {
					recordLogLine(fmt.Sprintln(err, " error!"))
					faultyInstances[instance.InstanceID] = faultyInstances[instance.InstanceID] + 1
					if faultyInstances[instance.InstanceID] > ReportingThreshold {
						// dealWithFaultyServer(instance, sess)
					}
					if faultyInstances[instance.InstanceID] > ReserThreshold {
						fmt.Printf("server %s is out, restarting\r\n", instance.InstanceID)
						restartingInstances[instance.InstanceID] = restartCounter{
							countingPoint: 1,
							restartPoint:  time.Now(),
						}
						if instance.RestartService() != nil {
							instance.HardRestartService()
						}
					}
				} else {
					if ok {
						// fmt.Println(" checked!")
						faultyInstances[instance.InstanceID] = 0
					} else {
						fmt.Println(instance.PrivateIPAddress, "failed!")
						faultyInstances[instance.InstanceID] = faultyInstances[instance.InstanceID] + 1
						if faultyInstances[instance.InstanceID] > ReportingThreshold {
							// dealWithFaultyServer(instance, sess)
						}
						if faultyInstances[instance.InstanceID] > ReserThreshold {
							fmt.Printf("server %s is out, restarting\r\n", instance.InstanceID)
							restartingInstances[instance.InstanceID] = restartCounter{
								countingPoint: 1,
								restartPoint:  time.Now(),
							}
							if instance.RestartService() != nil {
								instance.HardRestartService()
							}
						}
					}
					instance.FaultsCount = faultyInstances[instance.InstanceID]
				}
			}
			//fmt.Println("Round completed.")
			clientResponse.TimeStamp = time.Now()
			time.Sleep(TestDuration)
		}
	}
}
func recordLogLine(line string) {
	leToken := os.Getenv("LE_TOKEN")
	if leToken != "" {
		le, _ := le_go.Connect(leToken)
		le.Printf(line)
	}
}

func dealWithFaultyServer(faultyInstance *btrzaws.BetterezInstance, sess *session.Session) {
	leToken := os.Getenv("LE_TOKEN")
	if leToken != "" {
		le, _ := le_go.Connect(leToken)
		le.Printf("service %s is not responsive.", faultyInstance.InstanceID)
	}
	btrzaws.Notify(faultyInstance, sess)
}
