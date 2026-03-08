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

var dockerCmd = &cobra.Command{
	Use:   "docker [command] [args...]",
	Short: "Docker CLI with filtered output",
	Long: `Docker CLI with token-optimized output.

Specialized filters for common commands:
  - docker ps: Compact container listing
  - docker images: Image size summary
  - docker logs: Log deduplication
  - docker compose ps: Service status

Examples:
  tokman docker ps
  tokman docker images
  tokman docker logs <container>
  tokman docker compose ps`,
	RunE: runDocker,
}

func init() {
	rootCmd.AddCommand(dockerCmd)
}

func runDocker(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runDockerPassthrough(args)
	}

	switch args[0] {
	case "ps":
		return runDockerPs(args[1:])
	case "images":
		return runDockerImages(args[1:])
	case "logs":
		return runDockerLogs(args[1:])
	case "compose":
		if len(args) > 1 {
			switch args[1] {
			case "ps":
				return runComposePs(args[2:])
			case "logs":
				return runComposeLogs(args[2:])
			case "build":
				return runComposeBuild(args[2:])
			}
		}
	}
	return runDockerPassthrough(args)
}

func runDockerPassthrough(args []string) error {
	timer := tracking.Start()

	c := exec.Command("docker", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	fmt.Print(output)

	originalTokens := filter.EstimateTokens(output)
	timer.Track(fmt.Sprintf("docker %s", strings.Join(args, " ")), "tokman docker", originalTokens, originalTokens)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

func runDockerPs(args []string) error {
	timer := tracking.Start()

	// Get structured output
	c := exec.Command("docker", "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}")
	c.Env = os.Environ()

	var stdout bytes.Buffer
	c.Stdout = &stdout

	err := c.Run()
	output := stdout.String()

	if strings.TrimSpace(output) == "" {
		fmt.Println("🐳 0 containers")
		timer.Track("docker ps", "tokman docker ps", 0, 0)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := len(lines)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 %d containers:\n", count))

	for i, line := range lines {
		if i >= 15 {
			break
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 4 {
			id := parts[0]
			if len(id) > 12 {
				id = id[:12]
			}
			name := parts[1]
			shortImage := parts[3]
			if idx := strings.LastIndex(shortImage, "/"); idx != -1 {
				shortImage = shortImage[idx+1:]
			}
			ports := ""
			if len(parts) > 4 && parts[4] != "" {
				ports = fmt.Sprintf(" [%s]", compactPorts(parts[4]))
			}
			result.WriteString(fmt.Sprintf("  %s %s (%s)%s\n", id, name, shortImage, ports))
		}
	}

	if count > 15 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", count-15))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("docker ps", "tokman docker ps", originalTokens, filteredTokens)

	return err
}

func runDockerImages(args []string) error {
	timer := tracking.Start()

	c := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}\t{{.Size}}")
	c.Env = os.Environ()

	var stdout bytes.Buffer
	c.Stdout = &stdout

	err := c.Run()
	output := stdout.String()

	if strings.TrimSpace(output) == "" {
		fmt.Println("🐳 0 images")
		timer.Track("docker images", "tokman docker images", 0, 0)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := len(lines)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 %d images\n", count))

	for i, line := range lines {
		if i >= 15 {
			break
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 1 {
			image := parts[0]
			size := ""
			if len(parts) > 1 {
				size = fmt.Sprintf(" [%s]", parts[1])
			}
			if len(image) > 40 {
				image = "..." + image[len(image)-37:]
			}
			result.WriteString(fmt.Sprintf("  %s%s\n", image, size))
		}
	}

	if count > 15 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", count-15))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("docker images", "tokman docker images", originalTokens, filteredTokens)

	return err
}

func runDockerLogs(args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		fmt.Println("Usage: tokman docker logs <container>")
		return nil
	}

	container := args[0]
	c := exec.Command("docker", "logs", "--tail", "100", container)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterLogOutput(output)
	fmt.Printf("🐳 Logs for %s:\n%s", container, filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("docker logs %s", container), "tokman docker logs", originalTokens, filteredTokens)

	return err
}

func runComposePs(args []string) error {
	timer := tracking.Start()

	c := exec.Command("docker", "compose", "ps", "--format", "{{.Name}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
	c.Env = os.Environ()

	var stdout bytes.Buffer
	c.Stdout = &stdout

	err := c.Run()
	output := stdout.String()

	if strings.TrimSpace(output) == "" {
		fmt.Println("🐳 0 compose services")
		timer.Track("docker compose ps", "tokman docker compose ps", 0, 0)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := len(lines)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 %d compose services:\n", count))

	for i, line := range lines {
		if i >= 20 {
			break
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			name := parts[0]
			image := parts[1]
			status := parts[2]
			ports := ""
			if len(parts) > 3 && parts[3] != "" {
				ports = fmt.Sprintf(" [%s]", compactPorts(parts[3]))
			}
			shortImage := image
			if idx := strings.LastIndex(image, "/"); idx != -1 {
				shortImage = image[idx+1:]
			}
			result.WriteString(fmt.Sprintf("  %s (%s) %s%s\n", name, shortImage, status, ports))
		}
	}

	if count > 20 {
		result.WriteString(fmt.Sprintf("  ... +%d more\n", count-20))
	}

	filtered := result.String()
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("docker compose ps", "tokman docker compose ps", originalTokens, filteredTokens)

	return err
}

func runComposeLogs(args []string) error {
	timer := tracking.Start()

	c := exec.Command("docker", "compose", "logs", "--tail", "100")
	c.Args = append(c.Args, args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterLogOutput(output)
	fmt.Printf("🐳 Compose logs:\n%s", filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("docker compose logs", "tokman docker compose logs", originalTokens, filteredTokens)

	return err
}

func runComposeBuild(args []string) error {
	timer := tracking.Start()

	c := exec.Command("docker", "compose", "build")
	c.Args = append(c.Args, args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterComposeBuild(output)
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("docker compose build", "tokman docker compose build", originalTokens, filteredTokens)

	return err
}

func compactPorts(ports string) string {
	if ports == "" {
		return "-"
	}

	portNums := []string{}
	for _, p := range strings.Split(ports, ",") {
		p = strings.TrimSpace(p)
		if idx := strings.Index(p, "->"); idx != -1 {
			p = p[:idx]
		}
		if idx := strings.LastIndex(p, ":"); idx != -1 {
			p = p[idx+1:]
		}
		if p != "" {
			portNums = append(portNums, p)
		}
	}

	if len(portNums) <= 3 {
		return strings.Join(portNums, ", ")
	}
	return fmt.Sprintf("%s, ... +%d", strings.Join(portNums[:2], ", "), len(portNums)-2)
}

func filterLogOutput(output string) string {
	// Simple log deduplication - count repeated lines
	lines := strings.Split(output, "\n")
	var result strings.Builder
	prevLine := ""
	repeatCount := 0

	for _, line := range lines {
		if line == prevLine {
			repeatCount++
			continue
		}

		if repeatCount > 1 {
			result.WriteString(fmt.Sprintf("  [repeated %dx]\n", repeatCount))
		}
		repeatCount = 0

		if strings.TrimSpace(line) != "" {
			result.WriteString(line + "\n")
		}
		prevLine = line
	}

	if repeatCount > 1 {
		result.WriteString(fmt.Sprintf("  [repeated %dx]\n", repeatCount))
	}

	return result.String()
}

func filterComposeBuild(output string) string {
	var result strings.Builder

	// Find build summary
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Building") && strings.Contains(line, "FINISHED") {
			result.WriteString(fmt.Sprintf("🐳 %s\n", strings.TrimSpace(line)))
			break
		}
	}

	if result.Len() == 0 {
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "Building") {
				result.WriteString(fmt.Sprintf("🐳 %s\n", strings.TrimSpace(line)))
				break
			}
		}
	}

	if result.Len() == 0 {
		result.WriteString("🐳 Build:\n")
	}

	// Count services
	services := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "["); idx != -1 {
			if idx2 := strings.Index(line[idx:], "]"); idx2 != -1 {
				bracket := line[idx+1 : idx+idx2]
				parts := strings.Fields(bracket)
				if len(parts) > 0 {
					svc := parts[0]
					if svc != "" && svc != "+" {
						services[svc] = true
					}
				}
			}
		}
	}

	if len(services) > 0 {
		var names []string
		for name := range services {
			names = append(names, name)
		}
		result.WriteString(fmt.Sprintf("  Services: %s\n", strings.Join(names, ", ")))
	}

	return result.String()
}

// JSON structures for kubectl/aws
type K8sPodList struct {
	Items []K8sPod `json:"items"`
}

type K8sPod struct {
	Metadata K8sMetadata `json:"metadata"`
	Status   K8sPodStatus `json:"status"`
}

type K8sMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type K8sPodStatus struct {
	Phase             string                   `json:"phase"`
	ContainerStatuses []K8sContainerStatus `json:"containerStatuses"`
}

type K8sContainerStatus struct {
	Name        string `json:"name"`
	RestartCount int   `json:"restartCount"`
	State       map[string]interface{} `json:"state"`
}

type K8sServiceList struct {
	Items []K8sService `json:"items"`
}

type K8sService struct {
	Metadata K8sMetadata `json:"metadata"`
	Spec     K8sServiceSpec `json:"spec"`
}

type K8sServiceSpec struct {
	Type  string        `json:"type"`
	Ports []K8sServicePort `json:"ports"`
}

type K8sServicePort struct {
	Port       int    `json:"port"`
	TargetPort string `json:"targetPort"`
}

// Helper to unmarshal JSON
func unmarshalJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
