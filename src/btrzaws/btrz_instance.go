package btrzaws

import (
	"errors"
	"fmt"
	"logging"
	"net/http"
	"os"
	"sshconnector"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// BetterezInstance - aws representation, for betterez
type BetterezInstance struct {
	Environment            string
	Repository             string
	PrivateIPAddress       string
	PublicIPAddress        string
	BuildNumber            int
	KeyName                string
	InstanceName           string
	InstanceID             string
	PathName               string
	ServiceStatus          string
	ServiceStatusErrorCode string
	FaultsCount            int
	StatusCheck            time.Time
	AwsInstance            *ec2.Instance
}

const (
	// ConnectionTimeout - waiting time in which healthchceck should be back
	ConnectionTimeout = time.Duration(5 * time.Second)
)

// LoadFromAWSInstance - returns new BetterezInstance or an error
func LoadFromAWSInstance(instance *ec2.Instance) *BetterezInstance {
	result := &BetterezInstance{
		Environment:  GetTagValue(instance, "Environment"),
		Repository:   GetTagValue(instance, "Repository"),
		PathName:     GetTagValue(instance, "Path-Name"),
		InstanceName: GetTagValue(instance, "Name"),
		InstanceID:   *instance.InstanceId,
		KeyName:      *instance.KeyName,
		AwsInstance:  instance,
	}
	if instance.PublicIpAddress != nil {
		result.PublicIPAddress = *instance.PublicIpAddress
	}

	if instance.PrivateIpAddress != nil {
		result.PrivateIPAddress = *instance.PrivateIpAddress
	}
	buildNumber, err := strconv.Atoi(GetTagValue(instance, "Build-Number"))
	if err != nil {
		result.BuildNumber = 0
	} else {
		result.BuildNumber = buildNumber
	}
	return result
}

// GetHealthCheckString - Creates the healthcheck string based on the service name and address
func (instance *BetterezInstance) GetHealthCheckString() string {
	port := 3000
	var testURL string
	var testIPAddress string
	if instance.PublicIPAddress != "" {
		testIPAddress = instance.PublicIPAddress
	} else {
		testIPAddress = instance.PrivateIPAddress
	}
	if instance.PathName == "webhooks" || instance.PathName == "loyalty" {
		port = 4000
	}
	if instance.Repository == "connex2" {
		port = 22000
		testURL = fmt.Sprintf("http://%s:%d/healthcheck", testIPAddress, port)
	} else if instance.PathName != "/" {
		testURL = fmt.Sprintf("http://%s:%d/%s/healthcheck", testIPAddress, port, instance.PathName)
	} else {
		testURL = fmt.Sprintf("http://%s:%d/healthcheck", testIPAddress, port)
	}
	return testURL
}

// CheckIsnstanceHealth - checks instance health
func (instance *BetterezInstance) CheckIsnstanceHealth() (bool, error) {
	if instance == nil || instance.PrivateIPAddress == "" {
		return true, nil
	}
	httpClient := http.Client{
		Timeout: ConnectionTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := httpClient.Get(instance.GetHealthCheckString())
	instance.StatusCheck = time.Now()
	if err != nil {
		instance.ServiceStatus = "offline"
		instance.ServiceStatusErrorCode = fmt.Sprintf("%v", err)
		//log.Printf("Error %v healthcheck instance %s", err, instance.InstanceID)
		return false, err
	}
	defer resp.Body.Close()
	//log.Print("checking ", instance.Repository, "...")
	if resp.StatusCode > 0 && resp.StatusCode < 400 {
		instance.ServiceStatus = "online"
		instance.ServiceStatusErrorCode = ""
		return true, nil
	}
	return false, nil
}

// GetTagValue - return the tag value by name or an empty string
func (instance *BetterezInstance) GetTagValue(tagName string) string {
	return GetTagValue(instance.AwsInstance, tagName)
}

// GetKeysPath - return a string with the key path
func GetKeysPath() string {
	location := os.Getenv("SSH_KEYS_LOCATION")
	if !strings.HasSuffix(location, fmt.Sprintf("%c", os.PathSeparator)) {
		location += fmt.Sprintf("%c", os.PathSeparator)
	}
	return location
}

// RestartService - restart the service pointed by the tag
func (instance *BetterezInstance) RestartService() error {
	serviceName := instance.GetTagValue("Repository")
	if serviceName == "" {
		return errors.New("no service name found")
	}
	if GetKeysPath() == "" {
		return errors.New("no keys path")
	}
	//os.path
	keyFileLocation := fmt.Sprintf("%s%s.pem", GetKeysPath(), instance.KeyName)
	if _, err := os.Stat(keyFileLocation); os.IsNotExist(err) {
		return fmt.Errorf("%s key file doesn't exist", keyFileLocation)
	}
	sshConnection, err := sshconnector.CreateSSHSession(instance.PrivateIPAddress, "ubuntu", keyFileLocation, 22, sshconnector.UseKey)
	if err != nil {
		return err
	}
	err = sshConnection.Run(fmt.Sprintf("sudo service %s restart", serviceName))
	return err
}

// RestartServer - stop and start the aws vm in case it doesn't responde
func (instance *BetterezInstance) RestartServer() error {
	session, err := GetAWSSession()
	if err != nil {
		return err
	}
	ec2Service := ec2.New(session)
	_, err = ec2Service.StopInstances(&ec2.StopInstancesInput{
		DryRun: aws.Bool(false),
		InstanceIds: []*string{
			aws.String(instance.InstanceID),
		},
	})
	if err != nil {
		return err
	}
	processStatus := 0
	go func() {
		for {
			time.Sleep(10)
			output, err := ec2Service.DescribeInstances(&ec2.DescribeInstancesInput{
				DryRun: aws.Bool(false),
				InstanceIds: []*string{
					aws.String(instance.InstanceID),
				},
			})
			if err != nil {
				break
			}
			if *output.Reservations[0].Instances[0].State.Name == "stopped" && processStatus == 0 {
				logging.RecordLogLine(fmt.Sprintf("server %s stopped", instance.InstanceID))
				processStatus = 1
				ec2Service.StartInstances(&ec2.StartInstancesInput{
					DryRun: aws.Bool(false),
					InstanceIds: []*string{
						aws.String(instance.InstanceID),
					},
				})
			} else if processStatus == 1 && *output.Reservations[0].Instances[0].State.Name == "running" {
				logging.RecordLogLine(fmt.Sprintf("server %s is running", instance.InstanceID))
				break
			}
		}
	}()
	return nil
}
