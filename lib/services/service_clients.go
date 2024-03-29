package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"ServerBoi/cfg"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func RunServerBackup(server cfg.Server) string {
	var msg string

	server.ServiceInfo.Service.Name()

	if server.ServiceInfo.Service.Name() == "aws" {
		client := createSSMClient(server)

		msg = awsRunServerBackupCommand(client, server)

	} else {
		msg = "No provided service"
	}

	return msg
}

func awsRunServerBackupCommand(client ssm.Client, server cfg.Server) string {
	docName := "ServerBackup"
	instanceID := server.ServiceInfo.Service.Instance()

	//FIX Data Ingestion of ServerInfo params
	source := []string{
		server.Commands.BackupToS3.Source,
	}

	destination := []string{
		server.Commands.BackupToS3.Destination,
	}

	parameters := map[string][]string{
		"Source":      source,
		"Destination": destination,
	}

	sendInput := &ssm.SendCommandInput{
		DocumentName: &docName,
		InstanceIds: []string{
			instanceID,
		},
		Parameters: parameters,
	}

	commandResp, err := client.SendCommand(context.TODO(), sendInput)
	if err != nil {
		fmt.Println(err)
	}

	getInput := &ssm.GetCommandInvocationInput{
		CommandId:  commandResp.Command.CommandId,
		InstanceId: &instanceID,
	}

	var msg string

	for {
		time.Sleep(1 * time.Second)

		invocResp, err := client.GetCommandInvocation(context.TODO(), getInput)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(invocResp)

		if invocResp.Status == "Success" {

			msg = "Save has been backed up."

			break
		} else if invocResp.Status == "Failed" {

			msg = "Function has failed."

			break

		}

	}

	return msg

}

func GetServerCPU(server cfg.Server) string {
	var msg string

	if server.ServiceInfo.Service.Name() == "aws" {
		client := createSSMClient(server)

		msg = awsGetInstanceUtil(client, server)

	} else {
		msg = "No provided service"
	}

	return msg
}

func awsGetInstanceUtil(client ssm.Client, server cfg.Server) string {
	docName := "SystemUtilization"
	instanceID := server.ServiceInfo.Service.Instance()

	sendInput := &ssm.SendCommandInput{
		DocumentName: &docName,
		InstanceIds: []string{
			instanceID,
		},
	}

	commandResp, err := client.SendCommand(context.TODO(), sendInput)
	if err != nil {
		fmt.Println(err)
	}

	getInput := &ssm.GetCommandInvocationInput{
		CommandId:  commandResp.Command.CommandId,
		InstanceId: &instanceID,
	}

	var msg string

	for {
		time.Sleep(1 * time.Second)

		invocResp, err := client.GetCommandInvocation(context.TODO(), getInput)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(invocResp)

		if invocResp.Status == "Success" {

			msg = *invocResp.StandardOutputContent

			break
		} else if invocResp.Status == "Failed" {

			msg = "Function has failed"

			break

		}

	}

	return msg
}

//GetInstanceInfo | Returns instances info. Only AWS currently
func GetInstanceInfo(server cfg.Server) map[string]string {

	var instanceInfo map[string]string

	if server.ServiceInfo.Service.Name() == "aws" {
		client := createEc2Client(server)

		instanceInfo, err := awsDescribeInstance(client, server)
		if err != nil {
			fmt.Println(err)
		}

		return instanceInfo
	}

	return instanceInfo
}

func StartServer(server cfg.Server) bool {
	var success bool

	if server.ServiceInfo.Service.Name() == "aws" {
		client := createEc2Client(server)

		success = awsStartInstance(client, server)
	}

	return success

}

func StopServer(server cfg.Server) {
	if server.ServiceInfo.Service.Name() == "aws" {
		client := createEc2Client(server)

		awsStopInstance(client, server)
	}
}

func RebootServer(server cfg.Server) {
	if server.ServiceInfo.Service.Name() == "aws" {
		client := createEc2Client(server)

		awsRebootInstance(client, server)
	}
}

func awsRebootInstance(client ec2.Client, server cfg.Server) {
	instanceID := server.ServiceInfo.Service.Instance()

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
	instanceID := server.ServiceInfo.Service.Instance()

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

func awsStartInstance(client ec2.Client, server cfg.Server) bool {
	instanceID := server.ServiceInfo.Service.Instance()

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}

	var success bool

	_, err := client.StartInstances(context.TODO(), input)
	if err != nil {
		log.Println("Got an error retrieving starting EC2 instances:")
		log.Println(err)
		success = false
	} else {
		success = true
	}

	return success

}

func awsDescribeInstance(client ec2.Client, server cfg.Server) (map[string]string, error) {
	fmt.Println("Starting describe")

	instanceID := server.ServiceInfo.Service.Instance()

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

	creds := getRemoteCreds(server)

	region := server.ServiceInfo.Service.Geolocation()

	client := ec2.New(ec2.Options{
		Region:      region,
		Credentials: creds,
	})

	return *client
}

// CreateSSMClient creates a client to communicate with EC2
func createSSMClient(server cfg.Server) ssm.Client {

	creds := getRemoteCreds(server)

	region := server.ServiceInfo.Service.Geolocation()

	client := ssm.New(ssm.Options{
		Region:      region,
		Credentials: creds,
	})

	return *client
}

// CreateEc2Client creates a client to communicate with EC2
func createCloudwatchClient(server cfg.Server) cloudwatch.Client {

	creds := getRemoteCreds(server)

	region := server.ServiceInfo.Service.Geolocation()

	client := cloudwatch.New(cloudwatch.Options{
		Region:      region,
		Credentials: creds,
	})

	return *client
}

func getRemoteCreds(server cfg.Server) *aws.CredentialsCache {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("unable to load SDK config, %v", err)
	}

	region := server.ServiceInfo.Service.Geolocation()
	accountID := server.ServiceInfo.Service.Account()

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

	return creds

}
