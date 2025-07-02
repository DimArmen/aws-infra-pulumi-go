package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gopkg.in/yaml.v3"
)

// Config represents the infrastructure configuration
type Config struct {
	Environment string `yaml:"environment"`
	Customer    string `yaml:"customer"`
	// Add other fields as needed
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	stage := os.Args[1]

	switch stage {
	case "init":
		handleInit()
	case "vpc", "core", "apps":
		if len(os.Args) < 3 {
			log.Fatalf("Usage: cmd-deploy %s <pulumi-action> --config <file>", stage)
		}
		action := os.Args[2]
		configFile := getConfigFile()
		handleDeployStage(stage, action, configFile)
	default:
		log.Fatalf("Unknown command: %s", stage)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  cmd-deploy init --config <file>")
	fmt.Println("  cmd-deploy {vpc|core|apps} {up|down|preview} --config <file>")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  cmd-deploy init --config configs/sample-config.yaml")
	fmt.Println("  cmd-deploy vpc up --config configs/sample-config.yaml")
	fmt.Println("  cmd-deploy core preview --config configs/sample-config.yaml")
}

func getConfigFile() string {
	for i, arg := range os.Args {
		if arg == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	log.Fatal("--config flag is required")
	return ""
}

func handleInit() {
	fmt.Println("Initializing infrastructure...")

	// Load config to get bucket name
	configFile := getConfigFile()
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		log.Fatal("AWS_REGION environment variable must be set")
	}

	bucketName := fmt.Sprintf("pulumi-state-%s-%s", config.Environment, config.Customer)

	fmt.Printf("Creating S3 state bucket: %s\n", bucketName)
	if err := createS3Bucket(bucketName, region); err != nil {
		log.Fatalf("Failed to create S3 bucket: %v", err)
	}

	fmt.Printf("Configuring Pulumi backend: s3://%s\n", bucketName)
	if err := runCommand("pulumi", "login", fmt.Sprintf("s3://%s", bucketName)); err != nil {
		log.Fatalf("Failed to configure Pulumi backend: %v", err)
	}

	fmt.Println("Creating stacks...")
	stages := []string{"vpc", "core", "apps"}

	for _, stage := range stages {
		microstacks := getMicrostacksForStage(stage)

		for _, microstack := range microstacks {
			stackName := fmt.Sprintf("%s-%s-%s-%s", config.Customer, stage, microstack, region)
			fmt.Printf("Creating stack: %s\n", stackName)

			if err := runCommand("pulumi", "stack", "init", stackName); err != nil {
				fmt.Printf("Stack %s may already exist, continuing...\n", stackName)
			}
		}
	}

	fmt.Println("✅ Initialization complete!")
}

func handleDeployStage(stage, action, configFile string) {
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		log.Fatal("AWS_REGION environment variable must be set")
	}

	bucketName := fmt.Sprintf("pulumi-state-%s-%s", config.Environment, config.Customer)

	// Connect to Pulumi backend
	fmt.Printf("Logging into S3 backend: s3://%s\n", bucketName)
	if err := runCommand("pulumi", "login", fmt.Sprintf("s3://%s", bucketName)); err != nil {
		log.Fatalf("Failed to login to Pulumi backend: %v", err)
	}

	// Get microstacks for this stage
	microstacks := getMicrostacksForStage(stage)

	fmt.Printf("Deploying stage: %s with action: %s\n", stage, action)
	fmt.Printf("Microstacks to process: %s\n", strings.Join(microstacks, ", "))

	// Execute action on each microstack in order
	for _, microstack := range microstacks {
		stackName := fmt.Sprintf("%s-%s-%s-%s", config.Customer, stage, microstack, region)

		fmt.Printf("Processing microstack: %s (%s)\n", microstack, stackName)

		// Select the microstack
		if err := runCommand("pulumi", "stack", "select", stackName); err != nil {
			log.Fatalf("Failed to select stack: %v", err)
		}

		// Pass config to Pulumi program
		os.Setenv("CONFIG_FILE", configFile)
		if err := runCommand("pulumi", "config", "set", "microstack", microstack); err != nil {
			log.Fatalf("Failed to set microstack config: %v", err)
		}

		// Execute Pulumi action on this microstack
		var args []string
		if action == "up" || action == "down" {
			args = []string{action, "--yes"}
		} else {
			args = []string{action}
		}

		if err := runCommand("pulumi", args...); err != nil {
			log.Fatalf("Failed to run pulumi %s: %v", action, err)
		}

		fmt.Printf("✅ Completed %s %s\n", microstack, action)
	}

	fmt.Printf("✅ Successfully completed stage %s %s\n", stage, action)
}

func getMicrostacksForStage(stage string) []string {
	switch stage {
	case "vpc":
		return []string{"networking", "acls"}
	case "core":
		return []string{"s3", "route53", "rds", "eks", "opensearch", "cloudfront", "certificates"}
	case "apps":
		return []string{"eks-addons", "helm-charts", "storage-classes", "ingress-classes"}
	default:
		return []string{}
	}
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func createS3Bucket(bucketName, region string) error {
	ctx := context.TODO()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Check if bucket exists
	_, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		fmt.Printf("S3 bucket already exists: %s\n", bucketName)
		return nil
	}

	// Create bucket
	createBucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// For regions other than us-east-1, specify the LocationConstraint
	if region != "us-east-1" {
		createBucketInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err = s3Client.CreateBucket(ctx, createBucketInput)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Enable versioning
	_, err = s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}

	fmt.Printf("Created S3 bucket: %s\n", bucketName)
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
