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
	TerminateOnFault       string
	FaultsCount            int
	StatusCheck            time.Time
	AwsInstance            *ec2.Instance
	HealthcheckPath        string
	HelthcheckPort         int
	AutoScalingGroupName   string
}

const (
	ConnectionTimeout = time.Duration(5 * time.Second)
	StandradAPIPort   = 3000
)

func LoadFromAWSInstance(instance *ec2.Instance) *BetterezInstance {
	result := &BetterezInstance{
		Environment:          GetTagValue(instance, "Environment"),
		Repository:           GetTagValue(instance, "Repository"),
		PathName:             GetTagValue(instance, "Path-Name"),
		InstanceName:         GetTagValue(instance, "Name"),
		HealthcheckPath:      GetTagValue(instance, "Healtcheck-Path"),
		TerminateOnFault:     GetTagValue(instance, "Terminate on fault"),
		AutoScalingGroupName: GetTagValue(instance, "aws:autoscaling:groupName"),
		InstanceID:           *instance.InstanceId,
		KeyName:              *instance.KeyName,
		AwsInstance:          instance,
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

	result.HelthcheckPort, err = strconv.Atoi(GetTagValue(instance, "Healtcheck-Port"))
	if err != nil {
		result.HelthcheckPort = 0
	}
	return result
}

func (instance *BetterezInstance) IsInstanceOnAutoScalingGroup() bool {
	return instance.AutoScalingGroupName != ""
}
func (instance *BetterezInstance) ShouldTerminateOnFault() bool {
	if instance.TerminateOnFault == "yes" {
		return true
	}
	return false
}

func (instance *BetterezInstance) IsSameInstanceAs(compInstance *BetterezInstance) bool {
	if instance.BuildNumber != compInstance.BuildNumber {
		return false
	}
	if instance.Repository != compInstance.Repository {
		return false
	}
	if instance.PrivateIPAddress != compInstance.PrivateIPAddress {
		return false
	}
	if instance.InstanceID != compInstance.InstanceID {
		return false
	}
	return true
}

func (instance *BetterezInstance) GetHealthCheckString() string {
	port := instance.HelthcheckPort
	if instance.HelthcheckPort == 0 {
		port = StandradAPIPort
	}
	var testURL string
	var testIPAddress string
	if instance.PublicIPAddress != "" {
		testIPAddress = instance.PublicIPAddress
	} else {
		testIPAddress = instance.PrivateIPAddress
	}
	if isUsingElixir(instance.PathName) {
		port = getElixrPort()
	}
	if instance.Repository == "connex2" {
		port = getConnexPort()
		testURL = fmt.Sprintf("http://%s:%d/healthcheck", testIPAddress, port)
	} else if instance.PathName != "/" {
		testURL = fmt.Sprintf("http://%s:%d/%s/healthcheck", testIPAddress, port, instance.PathName)
	} else {
		testURL = fmt.Sprintf("http://%s:%d/%s", testIPAddress, port, instance.HealthcheckPath)
	}
	return testURL
}

func getElixrPort() int {
	return 4000
}

func getConnexPort() int {
	return 22000
}

func isUsingElixir(pathName string) bool {
	if pathName == "webhooks" || pathName == "liveseatmaps" ||
		pathName == "loyalty" ||
		pathName == "seatmaps" {
		return true
	}
	return false
}

// CheckInstanceHealth - checks instance health
func (instance *BetterezInstance) CheckInstanceHealth() (bool, error) {
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

func (instance *BetterezInstance) TerminateInstance() error {
	sess, err := GetAWSSession()
	if err != nil {
		return err
	}
	ec2Service := ec2.New(sess)
	_, err = ec2Service.TerminateInstances(
		&ec2.TerminateInstancesInput{
			DryRun: aws.Bool(false),
			InstanceIds: []*string{
				aws.String(instance.InstanceID),
			},
		},
	)

	return err
}

func (instance *BetterezInstance) RestartService() error {
	serviceName := instance.GetTagValue("Repository")
	if serviceName == "" {
		return errors.New("no service name found")
	}
	if GetKeysPath() == "" {
		return errors.New("no keys path")
	}

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
