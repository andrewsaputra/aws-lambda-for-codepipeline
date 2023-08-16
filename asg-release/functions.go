package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func PrintCodepipelineEvent(event CodePipelineEvent) {
	rawBytes, err := json.Marshal(event)
	if err != nil {
		fmt.Println("failed to print codepipeline event")
		return
	}

	str := string(rawBytes)
	fmt.Println(str)
}

func FetchArtifactObject(event CodePipelineEvent) (*bytes.Buffer, error) {
	artifactCredentials := event.JobDetails.Data.ArtifactCredentials
	cfgS3, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				*artifactCredentials.AccessKeyId,
				*artifactCredentials.SecretAccessKey,
				*artifactCredentials.SessionToken,
			),
		),
		config.WithRegion(userParameters.ArtifactAmiRegion),
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	inputArtifact := event.JobDetails.Data.InputArtifacts[0]
	s3Client := s3.NewFromConfig(cfgS3)
	s3Object, err := s3Client.GetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: inputArtifact.Location.S3Location.BucketName,
			Key:    inputArtifact.Location.S3Location.ObjectKey,
		},
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer s3Object.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(s3Object.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return buf, nil
}

func IdentifyAmiId(event CodePipelineEvent) (*string, error) {
	buf, err := FetchArtifactObject(event)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	zipReader, err := zip.NewReader(
		bytes.NewReader(buf.Bytes()),
		int64(buf.Len()),
	)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	for _, file := range zipReader.File {
		if file.Name != userParameters.ArtifactAmiFilename {
			continue
		}

		reader, err := file.Open()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer reader.Close()

		rawBytes, err := io.ReadAll(reader)
		contents := strings.Split(string(rawBytes), ":")
		amiId := strings.TrimSpace(contents[1])

		return &amiId, nil
	}

	return nil, nil
}
