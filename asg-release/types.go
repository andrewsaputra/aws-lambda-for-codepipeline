package main

import (
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
)

type CodePipelineEvent struct {
	JobDetails *types.JobDetails `json:"CodePipeline.job"`
}

type UserParameters struct {
	FunctionRegion      string
	AutoScalingGroup    string
	LaunchTemplateId    string
	ArtifactAmiFilename string
	ArtifactAmiRegion   string
}
