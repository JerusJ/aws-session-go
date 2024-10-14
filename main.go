package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	DefaultRegion = "us-west-2"
)

type inputArgs struct {
	mfaToken string
	duration int32
	profile  string
	credFile string
}

func main() {
	mfaToken := flag.String("mfa-token", "", "MFA token (required)")
	profile := flag.String("profile", "default", "AWS profile")
	duration := flag.Int("duration", 3600, "Duration in seconds")
	credFile := flag.String("creds-file", "", "Path to AWS credentials file")
	help := flag.Bool("help", false, "Show usage")
	flag.Parse()

	// Because ~ and $HOME is not expanded automatically if you do:
	// -creds-file=~/.aws/credentials, the '=' stops the shell from expanding
	var credFileAbs string
	if *credFile != "" {
		credFileAbs = getAbsShell(*credFile)
	}
	if !isFile(credFileAbs) {
		log.Fatalf("cannot find AWS credentials file at specified path: '%s'", credFileAbs)
	}

	ctx := context.Background()

	if *help {
		fmt.Println("Usage: go run main.go -mfa-token <mfa-token> -profile <profile> -duration <duration> -creds-file <path/to/credentials/file>")
		fmt.Println("Example: go run main.go -mfa-token 123456 -profile default -duration 3600 -creds-file /path/to/credentials/file")
		return
	}

	if *mfaToken == "" {
		fmt.Println("MFA token is required")
		fmt.Println("Usage: go run main.go -mfa-token <mfa-token> -profile <profile> -duration <duration> -creds-file <path/to/credentials/file>")
		fmt.Println("Example: go run main.go -mfa-token 123456 -profile default -duration 3600 -creds-file /path/to/credentials/file")
		return
	}

	fmt.Println("MFA Token: ", *mfaToken)
	fmt.Println("Profile: ", *profile)
	fmt.Println("Duration: ", *duration)
	fmt.Println("Config File: ", *credFile)

	if *duration < 900 || *duration > 43200 {
		log.Fatal("duration must be between 900 and 43200 seconds (15 minutes and 12 hours)")
	}

	sessionArgs := inputArgs{
		mfaToken: *mfaToken,
		profile:  *profile,
		duration: int32(*duration),
		credFile: credFileAbs,
	}

	var cfg aws.Config
	var err error
	if sessionArgs.credFile != "" {
		cfg, err = config.LoadDefaultConfig(
			ctx,
			config.WithSharedConfigProfile(sessionArgs.profile),
			config.WithSharedCredentialsFiles([]string{sessionArgs.credFile}),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(
			ctx,
			config.WithSharedConfigProfile(sessionArgs.profile),
		)
	}

	if err != nil {
		fmt.Println("Error loading configuration, ", err)
		return
	}

	// Below is how to retrieve credentials from the config and print them
	/*
		data, retErr := cfg.Credentials.Retrieve(ctx)
		if retErr != nil {
			fmt.Println("Error retrieving credentials, ", retErr)
			fmt.Println("Please ensure you have a default profile in your ~/.aws/credentials file at minimum. If you have a different profile, you can specify it as an argument to this program. Example: go run main.go 123456 my-profile 3600")
			return
		}

		// fmt.Println("Access Key: ", data.AccessKeyID)
		// fmt.Println("Secret Key: ", data.SecretAccessKey)
		// fmt.Println("Session Token: ", data.SessionToken)
		// fmt.Println("Region: ", cfg.Region)
	*/

	if cfg.Region == "" {
		cfg.Region = DefaultRegion
	}

	client := sts.NewFromConfig(cfg)
	idInput := &sts.GetCallerIdentityInput{}
	idResult, idErr := client.GetCallerIdentity(ctx, idInput)
	if idErr != nil {
		log.Fatalf("Error getting identity: %s", idErr)
	}

	serialNumber := strings.Replace(*idResult.Arn, "user", "mfa", 1)
	sessionInput := &sts.GetSessionTokenInput{
		DurationSeconds: &sessionArgs.duration,
		SerialNumber:    &serialNumber,
		TokenCode:       &sessionArgs.mfaToken,
	}

	sessionResult, sessionErr := client.GetSessionToken(ctx, sessionInput)
	if sessionErr != nil {
		log.Fatalln("Error getting session token: ", sessionErr)
	}

	/*
		fmt.Println("Session Access Key: ", *sessionResult.Credentials.AccessKeyId)
		fmt.Printf("Type: %T\n", *sessionResult.Credentials.AccessKeyId)
		fmt.Println("Session Secret Key: ", *sessionResult.Credentials.SecretAccessKey)
		fmt.Printf("Type: %T\n", *sessionResult.Credentials.SecretAccessKey)
		fmt.Println("Session Token: ", *sessionResult.Credentials.SessionToken)
		fmt.Printf("Type: %T\n", *sessionResult.Credentials.SessionToken)
		fmt.Println("Session Expiration: ", *sessionResult.Credentials.Expiration)
		fmt.Printf("Type: %T\n", *sessionResult.Credentials.Expiration)
	*/

	updateConfig(sessionArgs.credFile, sessionArgs.profile+"-session", *sessionResult.Credentials.AccessKeyId, *sessionResult.Credentials.SecretAccessKey, *sessionResult.Credentials.SessionToken)
}
