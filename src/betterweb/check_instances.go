package betterweb

import (
	"btrzaws"
	// "fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/bsphere/le_go"
)

const (
	// FaultThreshold - how many failed attempts before notification
	FaultThreshold = 3
	// ReserThreshold - how many failed attempts befor ssh reset\
	ReserThreshold = 5
	// TestDuration - time to wait between testing
	TestDuration = 28 * time.Second
)

func checkInstances(sess *session.Session, clientResponse *ClientResponse) {
	faultyInstances := make(map[string]int)
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
				//fmt.Println("")
				//fmt.Println("checking ", instance.PrivateIPAddress, "...")
				ok, err := instance.CheckIsnstanceHealth()
				if err != nil {
					//fmt.Println(err, " error!")
					faultyInstances[instance.InstanceID] = faultyInstances[instance.InstanceID] + 1
					if faultyInstances[instance.InstanceID] > FaultThreshold {
						dealWithFaultyServer(instance, sess)
					}
				} else {
					if ok {
						// fmt.Println(" checked!")
						faultyInstances[instance.InstanceID] = 0
					} else {
						// fmt.Println(instance.PrivateIPAddress, "failed!")
						faultyInstances[instance.InstanceID] = faultyInstances[instance.InstanceID] + 1
						if faultyInstances[instance.InstanceID] > FaultThreshold {
							dealWithFaultyServer(instance, sess)
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

func dealWithFaultyServer(faultyInstance *btrzaws.BetterezInstance, sess *session.Session) {
	leToken := os.Getenv("LE_TOKEN")
	if leToken != "" {
		le, _ := le_go.Connect(leToken)
		le.Printf("service %s is not responsive.", faultyInstance.InstanceID)
	}
	btrzaws.Notify(faultyInstance, sess)
}
