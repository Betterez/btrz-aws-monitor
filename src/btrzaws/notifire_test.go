package btrzaws

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

type fireBaseData struct {
	Authkey      string `json:"authkey"`
	DeviceHandle string `json:"device_handle"`
}

func TestFCMNotification(t *testing.T) {
	const fireBaseDataFile = "../../secrets/firebase.json"
	if _, err := os.Stat(fireBaseDataFile); os.IsNotExist(err) {
		t.SkipNow()
	}
	fileData, err := ioutil.ReadFile(fireBaseDataFile)
	if err != nil {
		t.Fatal("can't read firebase data file, ", err)
	}
	fbData := &fireBaseData{}
	json.Unmarshal(fileData, fbData)
	instance := &BetterezInstance{
		Repository:       "test-repo",
		Environment:      "production",
		InstanceID:       "123456789",
		PrivateIPAddress: "1.1.1.1",
	}
	NotifyByPush(instance, fbData.Authkey)

}
func TestSMSNotification(t *testing.T) {
	t.SkipNow()
	const fireBaseDataFile = "../../secrets/firebase.json"
	if _, err := os.Stat(fireBaseDataFile); os.IsNotExist(err) {
		t.SkipNow()
	}
	instance := &BetterezInstance{
		Repository:       "test-repo",
		Environment:      "production",
		InstanceID:       "123456789",
		PrivateIPAddress: "1.1.1.1",
	}
	sess, err := GetAWSSession()
	if err != nil {
		return
	}
	NotifyBySMS(instance, sess, "put full number here")
}
