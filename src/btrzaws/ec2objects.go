package btrzaws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// GetInstancesWithTags - return all instances with specific tags
func GetInstancesWithTags(awsSession *session.Session, tags []*AwsTag) ([]*ec2.Reservation, error) {
	var filters = []*ec2.Filter{}
	tagValues := []*string{}
	for _, tag := range tags {
		for _, tagValue := range tag.TagValues {
			tagValues = append(tagValues, aws.String(tagValue))
		}
		filters = append(filters,
			&ec2.Filter{Name: &tag.TagName,
				Values: tagValues,
			})
	}
	svc := ec2.New(awsSession)
	description := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	resp, err := svc.DescribeInstances(description)
	if err != nil {
		return nil, err
	}
	return resp.Reservations, nil
}
