package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
)

func checkELBs(sess *session.Session) {
	svc := elb.New(sess)
	allTags := []*elb.TagDescription{}
	resp, err := svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		return
	}
	params := &elb.DescribeTagsInput{}
	for _, elbDescription := range resp.LoadBalancerDescriptions {
		//fmt.Println(*elbDescription.LoadBalancerName)
		params.LoadBalancerNames = append(params.LoadBalancerNames, elbDescription.LoadBalancerName)
		if len(params.LoadBalancerNames) > 18 {
			break
		}
	}
	resp2, err := svc.DescribeTags(params)
	if err != nil {
		fmt.Print("error:", err, "\n\n")
	}
	allTags = append(allTags, resp2.TagDescriptions...)
	for _, tagDescription := range resp2.TagDescriptions {
		fmt.Println(*tagDescription.LoadBalancerName)
		for _, tag := range tagDescription.Tags {
			fmt.Println("\t", *tag.Key, *tag.Value)
		}
	}
}
