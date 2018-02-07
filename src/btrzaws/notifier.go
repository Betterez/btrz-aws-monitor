package btrzaws

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"net/http"
	"os"
	"time"
)

const (
	// FirebaseServerURL - google firebase url
	FirebaseServerURL = "https://fcm.googleapis.com/fcm/send"
	// ContentTypeJSON - http content type json
	ContentTypeJSON = "application/json"
	//ContentType -content type header
	ContentType = "Content-Type"
	// Authorization - Authorization token
	Authorization = "Authorization"
)

// Notify notify error in the instance
func Notify(instance *BetterezInstance, sess *session.Session) bool {
	if os.Getenv("PHONE_NUMBER") != "" {
		NotifyBySMS(instance, sess, os.Getenv("PHONE_NUMBER"))
	}
	if os.Getenv("FIREBASE_AUTHCODE") != "" {
		NotifyByPush(instance, os.Getenv("FIREBASE_AUTHCODE"))
	}
	return true
}

// NotifyBySMS - notify to a user by phone sms
func NotifyBySMS(instance *BetterezInstance, sess *session.Session, phoneNumber string) {
	notificationService := sns.New(sess)
	smsParams := &sns.SetSMSAttributesInput{
		Attributes: map[string]*string{
			"DefaultSenderID": aws.String("betterez"),
		}}
	notificationService.SetSMSAttributes(smsParams)
	notificationService.Publish(&sns.PublishInput{
		PhoneNumber: aws.String(phoneNumber),
		Message:     aws.String(fmt.Sprintf("Production server %s", instance.InstanceName)),
		Subject:     aws.String("betterez"),
	})
}

// NotifyByPush - push to firebase
func NotifyByPush(instance *BetterezInstance, serverAuthKey string) (bool, error) {
	payload := []byte(fmt.Sprintf(`{
		"priority":"HIGH",
		"notification":{
		"title"    :"server down",
		"body":"%s server not responding",
		"sound":"siren1"
		},
		"to":"/topics/alerts"
		}`, instance.Repository))
	req, err := http.NewRequest("POST", FirebaseServerURL, bytes.NewBuffer(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set(ContentType, ContentTypeJSON)
	req.Header.Set(Authorization, serverAuthKey)
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode > 399 {
		return false, nil
	}
	return true, nil
}
