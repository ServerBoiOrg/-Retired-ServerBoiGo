package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"ServerBoi/cfg"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

//GetInstanceInfo | Returns instances info. Only AWS currently
func GetInstanceInfo(server cfg.Server) map[string]string {

	var instanceInfo map[string]string

	if server.ServiceInfo["Service"] == "aws" {
		client := createEc2Client(server)

		instanceInfo, err := awsDescribeInstance(client, server)
		if err != nil {
			fmt.Println(err)
		}

		return instanceInfo
	}

	return instanceInfo
}

func StartServer(server cfg.Server) {
	if server.ServiceInfo["Service"] == "aws" {
		client := createEc2Client(server)

		awsStartInstance(client, server)
	}
}

func StopServer(server cfg.Server) {
	if server.ServiceInfo["Service"] == "aws" {
		client := createEc2Client(server)

		awsStopInstance(client, server)
	}
}

func RebootServer(server cfg.Server) {
	if server.ServiceInfo["Service"] == "aws" {
		client := createEc2Client(server)

		awsRebootInstance(client, server)
	}
}

func awsRebootInstance(client ec2.Client, server cfg.Server) {
	instanceID := server.ServiceInfo["InstanceID"]

	input := &ec2.RebootInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}

	_, err := client.RebootInstances(context.TODO(), input)
	if err != nil {
		fmt.Println("Got an error retrieving starting EC2 instances:")
		fmt.Println(err)
	}
}

func awsStopInstance(client ec2.Client, server cfg.Server) {
	instanceID := server.ServiceInfo["InstanceID"]

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}

	_, err := client.StopInstances(context.TODO(), input)
	if err != nil {
		fmt.Println("Got an error retrieving starting EC2 instances:")
		fmt.Println(err)
	}
}

func awsStartInstance(client ec2.Client, server cfg.Server) {
	instanceID := server.ServiceInfo["InstanceID"]

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}

	_, err := client.StartInstances(context.TODO(), input)
	if err != nil {
		fmt.Println("Got an error retrieving starting EC2 instances:")
		fmt.Println(err)
	}

}

func awsDescribeInstance(client ec2.Client, server cfg.Server) (map[string]string, error) {
	fmt.Println("Starting describe")

	instanceID := server.ServiceInfo["InstanceID"]

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}

	fmt.Println("Calling describe")
	resp, err := client.DescribeInstances(context.TODO(), input)
	if err != nil {
		fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
		fmt.Println(err)
	}

	instanceInfo := make(map[string]string)

	fmt.Println("Flipping through results")
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			fmt.Println("trying to assign IP")
			if i.PublicIpAddress != nil {
				instanceInfo["ip"] = *i.PublicIpAddress
			}
			fmt.Println("trying to assign instance type")
			instanceInfo["instanceType"] = string(i.InstanceType)
			fmt.Println("trying to assign state")
			instanceInfo["state"] = string(i.State.Name)
		}
	}

	if len(instanceInfo) != 0 {
		return instanceInfo, nil
	}

	return instanceInfo, errors.New("No instance found")
}

// CreateEc2Client creates a client to communicate with EC2
func createEc2Client(server cfg.Server) ec2.Client {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("unable to load SDK config, %v", err)
	}

	region := server.ServiceInfo["Region"]
	accountID := server.ServiceInfo["AccountID"]

	roleSession := fmt.Sprintf("ServerBoiGo-%v-%v-Session", accountID, region)
	roleArn := fmt.Sprintf("arn:aws:iam::%v:role/ServerBoiRole", accountID)

	stsClient := sts.NewFromConfig(cfg)

	input := &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSession,
	}

	newRole, err := stsClient.AssumeRole(context.TODO(), input)
	if err != nil {
		fmt.Println("Got an error assuming the role:")
		fmt.Println(err)
	}

	accessKey := newRole.Credentials.AccessKeyId
	secretKey := newRole.Credentials.SecretAccessKey
	sessionToken := newRole.Credentials.SessionToken

	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(*accessKey, *secretKey, *sessionToken))

	// value, err := creds.Retrieve(context.TODO())

	client := ec2.New(ec2.Options{
		Region:      region,
		Credentials: creds,
	})
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return *client
}
