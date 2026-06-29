package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// DockerServer provides Docker operations via MCP.
type DockerServer struct{}

func NewDockerServer() *DockerServer { return &DockerServer{} }

func (s *DockerServer) Name() string       { return "docker" }
func (s *DockerServer) Description() string { return "Docker operations: list containers, images, run, stop" }

func (s *DockerServer) Tools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "docker_list_containers",
			Description: "List Docker containers",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"all": map[string]interface{}{"type": "boolean", "description": "Include stopped containers"},
				},
			},
		},
		{
			Name:        "docker_list_images",
			Description: "List Docker images",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "docker_run",
			Description: "Run a Docker container",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"image":   map[string]interface{}{"type": "string", "description": "Image name"},
					"command": map[string]interface{}{"type": "string", "description": "Command to run"},
					"detach":  map[string]interface{}{"type": "boolean", "description": "Run in background"},
				},
				"required": []string{"image"},
			},
		},
		{
			Name:        "docker_stop",
			Description: "Stop a Docker container",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"container": map[string]interface{}{"type": "string", "description": "Container ID or name"},
				},
				"required": []string{"container"},
			},
		},
	}
}

func (s *DockerServer) HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	switch params.Name {
	case "docker_list_containers":
		return s.listContainers(params.Arguments)
	case "docker_list_images":
		return s.listImages()
	case "docker_run":
		return s.runContainer(params.Arguments)
	case "docker_stop":
		return s.stopContainer(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *DockerServer) listContainers(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	cmdArgs := []string{"ps"}
	if all, ok := args["all"].(bool); ok && all {
		cmdArgs = append(cmdArgs, "-a")
	}
	cmdArgs = append(cmdArgs, "--format", "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}")

	output, err := exec.Command("docker", cmdArgs...).CombinedOutput()
	if err != nil {
		return toolError(fmt.Sprintf("docker ps failed: %v\n%s", err, string(output))), nil
	}

	return toolSuccess(string(output)), nil
}

func (s *DockerServer) listImages() (*protocol.ToolCallResult, error) {
	output, err := exec.Command("docker", "images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.Size}}").CombinedOutput()
	if err != nil {
		return toolError(fmt.Sprintf("docker images failed: %v", err)), nil
	}
	return toolSuccess(string(output)), nil
}

func (s *DockerServer) runContainer(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	image, _ := args["image"].(string)
	if image == "" {
		return toolError("image is required"), nil
	}

	cmdArgs := []string{"run"}
	if detach, ok := args["detach"].(bool); ok && detach {
		cmdArgs = append(cmdArgs, "-d")
	}
	cmdArgs = append(cmdArgs, image)
	if command, ok := args["command"].(string); ok && command != "" {
		cmdArgs = append(cmdArgs, strings.Fields(command)...)
	}

	output, err := exec.Command("docker", cmdArgs...).CombinedOutput()
	if err != nil {
		return toolError(fmt.Sprintf("docker run failed: %v\n%s", err, string(output))), nil
	}

	return toolSuccess(string(output)), nil
}

func (s *DockerServer) stopContainer(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	container, _ := args["container"].(string)
	if container == "" {
		return toolError("container is required"), nil
	}

	output, err := exec.Command("docker", "stop", container).CombinedOutput()
	if err != nil {
		return toolError(fmt.Sprintf("docker stop failed: %v\n%s", err, string(output))), nil
	}

	return toolSuccess(fmt.Sprintf("Container stopped: %s", container)), nil
}
