package btrzaws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// GetAWSSession -  creates an aws session
func GetAWSSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errors.New("can't create aws session")
	}
	return sess, nil
}
