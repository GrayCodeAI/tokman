package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var awsCmd = &cobra.Command{
	Use:   "aws [service] [command] [args...]",
	Short: "AWS CLI with filtered output",
	Long: `AWS CLI with token-optimized output.

Specialized filters for common services:
  - sts get-caller-identity: Compact identity
  - ec2 describe-instances: Instance summary
  - ecs list/describe-services: Service status
  - rds describe-db-instances: DB summary
  - cloudformation list/describe-stacks: Stack status

Examples:
  tokman aws sts get-caller-identity
  tokman aws ec2 describe-instances
  tokman aws s3 ls`,
	RunE: runAws,
}

func init() {
	rootCmd.AddCommand(awsCmd)
}

func runAws(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runAwsPassthrough(args)
	}

	service := args[0]
	var subcommand string
	if len(args) > 1 {
		subcommand = args[1]
	}

	switch {
	case service == "sts" && subcommand == "get-caller-identity":
		return runAwsStsIdentity(args[2:])
	case service == "s3" && subcommand == "ls":
		return runAwsS3Ls(args[2:])
	case service == "ec2" && subcommand == "describe-instances":
		return runAwsEc2Describe(args[2:])
	case service == "ecs" && subcommand == "list-services":
		return runAwsEcsListServices(args[2:])
	case service == "ecs" && subcommand == "describe-services":
		return runAwsEcsDescribeServices(args[2:])
	case service == "rds" && subcommand == "describe-db-instances":
		return runAwsRdsDescribe(args[2:])
	case service == "cloudformation" && subcommand == "list-stacks":
		return runAwsCfnListStacks(args[2:])
	case service == "cloudformation" && subcommand == "describe-stacks":
		return runAwsCfnDescribeStacks(args[2:])
	default:
		return runAwsPassthrough(args)
	}
}

func runAwsPassthrough(args []string) error {
	timer := tracking.Start()

	c := exec.Command("aws", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	fmt.Print(output)

	originalTokens := filter.EstimateTokens(output)
	timer.Track(fmt.Sprintf("aws %s", strings.Join(args, " ")), "tokman aws", originalTokens, originalTokens)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

func runAwsJson(subcmd []string, extraArgs []string) (string, string, error) {
	args := subcmd
	args = append(args, extraArgs...)
	args = append(args, "--output", "json")

	c := exec.Command("aws", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	return stdout.String(), stderr.String(), err
}

func runAwsStsIdentity(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"sts", "get-caller-identity"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
	}
	if unmarshalJSON(stdout, &identity) != nil {
		fmt.Print(stdout)
		return nil
	}

	filtered := fmt.Sprintf("AWS: %s %s\n", identity.Account, identity.Arn)
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws sts get-caller-identity", "tokman aws sts", originalTokens, filteredTokens)

	return nil
}

func runAwsS3Ls(extraArgs []string) error {
	timer := tracking.Start()

	args := []string{"s3", "ls"}
	args = append(args, extraArgs...)

	c := exec.Command("aws", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()

	if err != nil {
		fmt.Print(stderr.String())
		return err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	total := len(lines)

	var result strings.Builder
	for i, line := range lines {
		if i >= 20 {
			break
		}
		result.WriteString(line + "\n")
	}

	if total > 20 {
		result.WriteString(fmt.Sprintf("... +%d more items\n", total-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws s3 ls", "tokman aws s3 ls", originalTokens, filteredTokens)

	return nil
}

func runAwsEc2Describe(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"ec2", "describe-instances"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var ec2Resp struct {
		Reservations []struct {
			Instances []struct {
				InstanceId       string `json:"InstanceId"`
				InstanceType     string `json:"InstanceType"`
				PrivateIpAddress string `json:"PrivateIpAddress"`
				State            struct {
					Name string `json:"Name"`
				} `json:"State"`
				Tags []struct {
					Key   string `json:"Key"`
					Value string `json:"Value"`
				} `json:"Tags"`
			} `json:"Instances"`
		} `json:"Reservations"`
	}

	if unmarshalJSON(stdout, &ec2Resp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var instances []string
	for _, res := range ec2Resp.Reservations {
		for _, inst := range res.Instances {
			name := "-"
			for _, tag := range inst.Tags {
				if tag.Key == "Name" {
					name = tag.Value
					break
				}
			}
			ip := inst.PrivateIpAddress
			if ip == "" {
				ip = "-"
			}
			instances = append(instances, fmt.Sprintf("%s %s %s %s (%s)",
				inst.InstanceId, inst.State.Name, inst.InstanceType, ip, name))
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("EC2: %d instances\n", len(instances)))

	for i, inst := range instances {
		if i >= 20 {
			break
		}
		result.WriteString(fmt.Sprintf("  %s\n", inst))
	}

	if len(instances) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(instances)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws ec2 describe-instances", "tokman aws ec2", originalTokens, filteredTokens)

	return nil
}

func runAwsEcsListServices(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"ecs", "list-services"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var ecsResp struct {
		ServiceArns []string `json:"serviceArns"`
	}

	if unmarshalJSON(stdout, &ecsResp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var services []string
	for _, arn := range ecsResp.ServiceArns {
		// Extract name from ARN
		short := arn
		if idx := strings.LastIndex(arn, "/"); idx != -1 {
			short = arn[idx+1:]
		}
		services = append(services, short)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ECS: %d services\n", len(services)))

	for i, svc := range services {
		if i >= 20 {
			break
		}
		result.WriteString(fmt.Sprintf("  %s\n", svc))
	}

	if len(services) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(services)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws ecs list-services", "tokman aws ecs", originalTokens, filteredTokens)

	return nil
}

func runAwsEcsDescribeServices(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"ecs", "describe-services"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var ecsResp struct {
		Services []struct {
			ServiceName  string `json:"serviceName"`
			Status       string `json:"status"`
			RunningCount int    `json:"runningCount"`
			DesiredCount int    `json:"desiredCount"`
			LaunchType   string `json:"launchType"`
		} `json:"services"`
	}

	if unmarshalJSON(stdout, &ecsResp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ECS: %d services\n", len(ecsResp.Services)))

	for i, svc := range ecsResp.Services {
		if i >= 20 {
			break
		}
		launch := svc.LaunchType
		if launch == "" {
			launch = "FARGATE"
		}
		result.WriteString(fmt.Sprintf("  %s %s %d/%d (%s)\n",
			svc.ServiceName, svc.Status, svc.RunningCount, svc.DesiredCount, launch))
	}

	if len(ecsResp.Services) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(ecsResp.Services)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws ecs describe-services", "tokman aws ecs", originalTokens, filteredTokens)

	return nil
}

func runAwsRdsDescribe(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"rds", "describe-db-instances"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var rdsResp struct {
		DBInstances []struct {
			DBInstanceIdentifier string `json:"DBInstanceIdentifier"`
			Engine               string `json:"Engine"`
			EngineVersion        string `json:"EngineVersion"`
			DBInstanceClass      string `json:"DBInstanceClass"`
			DBInstanceStatus     string `json:"DBInstanceStatus"`
		} `json:"DBInstances"`
	}

	if unmarshalJSON(stdout, &rdsResp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("RDS: %d instances\n", len(rdsResp.DBInstances)))

	for i, db := range rdsResp.DBInstances {
		if i >= 20 {
			break
		}
		result.WriteString(fmt.Sprintf("  %s %s %s %s %s\n",
			db.DBInstanceIdentifier, db.Engine, db.EngineVersion, db.DBInstanceClass, db.DBInstanceStatus))
	}

	if len(rdsResp.DBInstances) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(rdsResp.DBInstances)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws rds describe-db-instances", "tokman aws rds", originalTokens, filteredTokens)

	return nil
}

func runAwsCfnListStacks(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"cloudformation", "list-stacks"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var cfnResp struct {
		StackSummaries []struct {
			StackName    string `json:"StackName"`
			StackStatus  string `json:"StackStatus"`
			LastUpdatedTime string `json:"LastUpdatedTime"`
			CreationTime string `json:"CreationTime"`
		} `json:"StackSummaries"`
	}

	if unmarshalJSON(stdout, &cfnResp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("CloudFormation: %d stacks\n", len(cfnResp.StackSummaries)))

	for i, stack := range cfnResp.StackSummaries {
		if i >= 20 {
			break
		}
		date := stack.LastUpdatedTime
		if date == "" {
			date = stack.CreationTime
		}
		// Truncate ISO date
		if len(date) > 10 {
			date = date[:10]
		}
		result.WriteString(fmt.Sprintf("  %s %s %s\n", stack.StackName, stack.StackStatus, date))
	}

	if len(cfnResp.StackSummaries) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(cfnResp.StackSummaries)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws cloudformation list-stacks", "tokman aws cfn", originalTokens, filteredTokens)

	return nil
}

func runAwsCfnDescribeStacks(extraArgs []string) error {
	timer := tracking.Start()

	stdout, stderr, err := runAwsJson([]string{"cloudformation", "describe-stacks"}, extraArgs)
	if err != nil {
		fmt.Print(stderr)
		return err
	}

	var cfnResp struct {
		Stacks []struct {
			StackName    string `json:"StackName"`
			StackStatus  string `json:"StackStatus"`
			LastUpdatedTime string `json:"LastUpdatedTime"`
			CreationTime string `json:"CreationTime"`
			Outputs []struct {
				OutputKey   string `json:"OutputKey"`
				OutputValue string `json:"OutputValue"`
			} `json:"Outputs"`
		} `json:"Stacks"`
	}

	if unmarshalJSON(stdout, &cfnResp) != nil {
		fmt.Print(stdout)
		return nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("CloudFormation: %d stacks\n", len(cfnResp.Stacks)))

	for i, stack := range cfnResp.Stacks {
		if i >= 20 {
			break
		}
		date := stack.LastUpdatedTime
		if date == "" {
			date = stack.CreationTime
		}
		if len(date) > 10 {
			date = date[:10]
		}
		result.WriteString(fmt.Sprintf("  %s %s %s\n", stack.StackName, stack.StackStatus, date))

		for _, out := range stack.Outputs {
			result.WriteString(fmt.Sprintf("    %s=%s\n", out.OutputKey, out.OutputValue))
		}
	}

	if len(cfnResp.Stacks) > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(cfnResp.Stacks)-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(stdout)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("aws cloudformation describe-stacks", "tokman aws cfn", originalTokens, filteredTokens)

	return nil
}

// Helper to detect JSON (for json_cmd integration)
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
