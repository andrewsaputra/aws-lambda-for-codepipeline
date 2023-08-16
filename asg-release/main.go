package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2type "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var userParameters UserParameters
var initCompleted bool

var ec2Client *ec2.Client
var codepipelineClient *codepipeline.Client
var autoscalingClient *autoscaling.Client

func init() {
	fmt.Println("init() invoked")

	userParameters = UserParameters{
		FunctionRegion:      os.Getenv("AWS_REGION"),
		AutoScalingGroup:    os.Getenv("func_AutoScalingGroup"),
		LaunchTemplateId:    os.Getenv("func_LaunchTemplateId"),
		ArtifactAmiFilename: os.Getenv("func_ArtifactAmiFilename"),
		ArtifactAmiRegion:   os.Getenv("func_ArtifactAmiRegion"),
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(userParameters.FunctionRegion),
	)
	if err != nil {
		log.Fatal(err)
	}

	ec2Client = ec2.NewFromConfig(cfg)
	codepipelineClient = codepipeline.NewFromConfig(cfg)
	autoscalingClient = autoscaling.NewFromConfig(cfg)

	initCompleted = true
}

/**
 *	Reference : https://docs.aws.amazon.com/codepipeline/latest/userguide/action-reference-Lambda.html
 *
 *	Process Sequence :
 *	1. Fetch artifact object from codepipeline bucket using temporary credentials
 *	2. Identify AMI ID from the relevant artifact contents
 *	3. Get latest version of target launch template
 *	4. Create new launch template version with updated ImageId
 *	5. Update default version of target launch template
 *	6. Trigger instance refresh on target auto scaling group
 */
func HandleRequest(ctx context.Context, event CodePipelineEvent) (*codepipeline.PutJobSuccessResultOutput, error) {
	if !initCompleted {
		return nil, fmt.Errorf("init() failed to complete")
	}

	PrintCodepipelineEvent(event)

	newAmiId, err := IdentifyAmiId(event)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	describeVersionsResult, err := ec2Client.DescribeLaunchTemplateVersions(
		context.Background(),
		&ec2.DescribeLaunchTemplateVersionsInput{
			LaunchTemplateId: aws.String(userParameters.LaunchTemplateId),
			Versions:         []string{"$Latest"},
		},
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	currentVersionData := describeVersionsResult.LaunchTemplateVersions[0].LaunchTemplateData
	createVersionResult, err := ec2Client.CreateLaunchTemplateVersion(
		context.Background(),
		&ec2.CreateLaunchTemplateVersionInput{
			LaunchTemplateId: aws.String(userParameters.LaunchTemplateId),
			LaunchTemplateData: &ec2type.RequestLaunchTemplateData{
				ImageId:          newAmiId,
				EbsOptimized:     currentVersionData.EbsOptimized,
				InstanceType:     currentVersionData.InstanceType,
				UserData:         currentVersionData.UserData,
				SecurityGroupIds: currentVersionData.SecurityGroupIds,

				IamInstanceProfile: &ec2type.LaunchTemplateIamInstanceProfileSpecificationRequest{
					Arn: currentVersionData.IamInstanceProfile.Arn,
				},

				CreditSpecification: &ec2type.CreditSpecificationRequest{
					CpuCredits: currentVersionData.CreditSpecification.CpuCredits,
				},
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	newVersion := strconv.FormatInt(*createVersionResult.LaunchTemplateVersion.VersionNumber, 10)
	_, err = ec2Client.ModifyLaunchTemplate(
		context.Background(),
		&ec2.ModifyLaunchTemplateInput{
			LaunchTemplateId: aws.String(userParameters.LaunchTemplateId),
			DefaultVersion:   aws.String(newVersion),
		},
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	refreshInstanceRes, err := autoscalingClient.StartInstanceRefresh(
		context.Background(),
		&autoscaling.StartInstanceRefreshInput{
			AutoScalingGroupName: aws.String(userParameters.AutoScalingGroup),
		},
	)
	if err != nil {
		fmt.Printf("skipping instance refresh due to error : %v \n", err.Error())
	}

	fmt.Printf("asg instance refresh id : %v \n", refreshInstanceRes.InstanceRefreshId)

	return codepipelineClient.PutJobSuccessResult(
		context.Background(),
		&codepipeline.PutJobSuccessResultInput{
			JobId: event.JobDetails.Id,
		},
	)
}

func main() {
	//rawBytes, _ := os.ReadFile("sample-event.json")
	//var evt CodePipelineEvent
	//json.Unmarshal(rawBytes, &evt)
	//HandleRequest(context.Background(), evt)

	lambda.Start(HandleRequest)
}
