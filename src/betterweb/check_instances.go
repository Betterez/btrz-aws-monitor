package betterweb

import (
	"btrzaws"
	"fmt"
	"logging"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	ReportingThreshold              = 3
	RestartThreshold                = 3
	TestDuration                    = 8 * time.Second
	SoftRestartDuration             = time.Second * 45
	HardRestartDuration             = time.Minute * 7
	NotificationResetDuration       = time.Hour * 1
	ServerAliveDurationNotification = time.Minute * 10
	InitializationDuration          = HardRestartDuration
)

type restartCounter struct {
	restartCheckpoint time.Time
	countingPoint     int
}

func notifyInstaneFailureStatus(faultyInstance *btrzaws.BetterezInstance, sess *session.Session) {
	logging.RecordLogLine(fmt.Sprintf("instance %s failure notice was sent. repo: %s", faultyInstance.InstanceID, faultyInstance.Repository))
	btrzaws.Notify(faultyInstance, sess)
}

func isThisInstanceStillStarting(instanceID string, listing *map[string]restartCounter) bool {
	if (*listing)[instanceID].countingPoint != 0 {
		if time.Now().Before((*listing)[instanceID].restartCheckpoint) {
			return true
		}
	}
	return false
}

func isThisInstanceJustCreated(instance *btrzaws.BetterezInstance) bool {
	if (instance.AwsInstance.LaunchTime.Add(InitializationDuration)).After(time.Now()) {
		return true
	}
	return false
}
