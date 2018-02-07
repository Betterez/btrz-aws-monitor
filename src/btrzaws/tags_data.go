package btrzaws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

// AwsTag simple tag struct
type AwsTag struct {
	TagName   string
	TagValues []string
}

// New - create new tag
func New() *AwsTag {
	result := &AwsTag{TagName: "", TagValues: []string{}}
	return result
}

// NewWithValues - create tag with values
func NewWithValues(name, value string) *AwsTag {
	result := &AwsTag{TagName: name, TagValues: []string{value}}
	return result
}

// GetTagValue - get tag value
func GetTagValue(instance *ec2.Instance, tagName string) string {
	var instanceName string
	for _, tag := range instance.Tags {
		if *tag.Key == tagName {
			instanceName = *tag.Value
		}
	}
	return instanceName
}
