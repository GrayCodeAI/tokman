package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var kubectlCmd = &cobra.Command{
	Use:   "kubectl [command] [args...]",
	Short: "Kubernetes CLI with filtered output",
	Long: `Kubernetes CLI with token-optimized output.

Specialized filters for common commands:
  - kubectl get pods: Pod status summary
  - kubectl get services: Service listing
  - kubectl logs: Log deduplication

Examples:
  tokman kubectl get pods
  tokman kubectl get pods -n production
  tokman kubectl get services
  tokman kubectl logs <pod>`,
	RunE: runKubectl,
}

func init() {
	rootCmd.AddCommand(kubectlCmd)
}

func runKubectl(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runKubectlPassthrough(args)
	}

	if args[0] == "get" && len(args) > 1 {
		switch args[1] {
		case "pods", "pod", "po":
			return runKubectlPods(args[2:])
		case "services", "service", "svc":
			return runKubectlServices(args[2:])
		}
	}

	if args[0] == "logs" && len(args) > 1 {
		return runKubectlLogs(args[1:])
	}

	return runKubectlPassthrough(args)
}

func runKubectlPassthrough(args []string) error {
	timer := tracking.Start()

	c := exec.Command("kubectl", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	fmt.Print(output)

	originalTokens := filter.EstimateTokens(output)
	timer.Track(fmt.Sprintf("kubectl %s", strings.Join(args, " ")), "tokman kubectl", originalTokens, originalTokens)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

func runKubectlPods(args []string) error {
	timer := tracking.Start()

	// Build command with JSON output
	kargs := []string{"get", "pods", "-o", "json"}
	kargs = append(kargs, args...)

	c := exec.Command("kubectl", kargs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	raw := stdout.String()

	if err != nil {
		fmt.Print(stderr.String())
		return err
	}

	var podList K8sPodList
	if unmarshalJSON(raw, &podList) != nil || len(podList.Items) == 0 {
		fmt.Println("☸️  No pods found")
		timer.Track("kubectl get pods", "tokman kubectl pods", filter.EstimateTokens(raw), 0)
		return nil
	}

	pods := podList.Items
	running, pending, failed, restartsTotal := 0, 0, 0, 0
	var issues []string

	for _, pod := range pods {
		ns := pod.Metadata.Namespace
		name := pod.Metadata.Name
		phase := pod.Status.Phase

		for _, cs := range pod.Status.ContainerStatuses {
			restartsTotal += cs.RestartCount
		}

		switch phase {
		case "Running":
			running++
		case "Pending":
			pending++
			issues = append(issues, fmt.Sprintf("%s/%s Pending", ns, name))
		case "Failed", "Error":
			failed++
			issues = append(issues, fmt.Sprintf("%s/%s %s", ns, name, phase))
		}
	}

	var parts []string
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d ✓", running))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d ✗", failed))
	}
	if restartsTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d restarts", restartsTotal))
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("☸️  %d pods: %s\n", len(pods), strings.Join(parts, ", ")))

	if len(issues) > 0 {
		result.WriteString("⚠️  Issues:\n")
		for i, issue := range issues {
			if i >= 10 {
				break
			}
			result.WriteString(fmt.Sprintf("  %s\n", issue))
		}
		if len(issues) > 10 {
			result.WriteString(fmt.Sprintf("  ... +%d more\n", len(issues)-10))
		}
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("kubectl get pods", "tokman kubectl pods", originalTokens, filteredTokens)

	return nil
}

func runKubectlServices(args []string) error {
	timer := tracking.Start()

	// Build command with JSON output
	kargs := []string{"get", "services", "-o", "json"}
	kargs = append(kargs, args...)

	c := exec.Command("kubectl", kargs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	raw := stdout.String()

	if err != nil {
		fmt.Print(stderr.String())
		return err
	}

	var svcList K8sServiceList
	if unmarshalJSON(raw, &svcList) != nil || len(svcList.Items) == 0 {
		fmt.Println("☸️  No services found")
		timer.Track("kubectl get services", "tokman kubectl services", filter.EstimateTokens(raw), 0)
		return nil
	}

	services := svcList.Items

	var result strings.Builder
	result.WriteString(fmt.Sprintf("☸️  %d services:\n", len(services)))

	for i, svc := range services {
		if i >= 15 {
			break
		}
		ns := svc.Metadata.Namespace
		name := svc.Metadata.Name
		svcType := svc.Spec.Type
		if svcType == "" {
			svcType = "ClusterIP"
		}

		var ports []string
		for _, p := range svc.Spec.Ports {
			port := p.Port
			target := p.TargetPort
			if target == "" || target == fmt.Sprintf("%d", port) {
				ports = append(ports, fmt.Sprintf("%d", port))
			} else {
				ports = append(ports, fmt.Sprintf("%d→%s", port, target))
			}
		}

		result.WriteString(fmt.Sprintf("  %s/%s %s [%s]\n", ns, name, svcType, strings.Join(ports, ",")))
	}

	if len(services) > 15 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", len(services)-15))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("kubectl get services", "tokman kubectl services", originalTokens, filteredTokens)

	return nil
}

func runKubectlLogs(args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		fmt.Println("Usage: tokman kubectl logs <pod>")
		return nil
	}

	pod := args[0]
	kargs := []string{"logs", "--tail", "100", pod}
	kargs = append(kargs, args[1:]...)

	c := exec.Command("kubectl", kargs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterLogOutput(output)
	fmt.Printf("☸️  Logs for %s:\n%s", pod, filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("kubectl logs %s", pod), "tokman kubectl logs", originalTokens, filteredTokens)

	return err
}
