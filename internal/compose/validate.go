package compose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/compose-spec/compose-go/v2/validation"
)

// validateProject validates a compose project against the compose specification.
// It runs compose-go's schema validation, which is deferred during initial loading
// to allow setting the project name from the directory.
func validateProject(ctx context.Context, project *types.Project) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate nil project
	if project == nil {
		return &validationError{message: "project is not defined"}
	}

	// Run compose-go validation using the types.Project marshaling.
	// The validation package works on map[string]any, so we marshal the project to JSON
	// and then validate it as a map structure.
	projectJSON, err := project.MarshalJSON()
	if err != nil {
		return &validationError{message: "failed to marshal project", cause: err}
	}

	// Parse JSON back into a map for compose-go's validation
	var projectMap map[string]any
	if err := json.Unmarshal(projectJSON, &projectMap); err != nil {
		return &validationError{message: "failed to unmarshal project", cause: err}
	}

	// Run compose-go's schema validation
	if err := validation.Validate(projectMap); err != nil {
		return &validationError{message: err.Error(), cause: err}
	}

	return nil
}

// validateQuadletCompatibility checks if a compose project can be converted to podman-systemd quadlet units.
// It validates that services have the required images and don't use incompatible features.
func validateQuadletCompatibility(ctx context.Context, project *types.Project) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if project == nil {
		return &quadletCompatibilityError{message: "project is nil"}
	}

	// Check each service for quadlet compatibility
	for serviceName, service := range project.Services {
		if err := validateServiceQuadletCompatibility(serviceName, service); err != nil {
			return err
		}
	}

	// Check for unsupported volume drivers
	for volumeName, vol := range project.Volumes {
		if vol.Driver != "" && vol.Driver != "local" {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("volume %q uses unsupported driver %q; only 'local' driver is supported", volumeName, vol.Driver),
			}
		}
	}

	// Check for unsupported network drivers
	for networkName, net := range project.Networks {
		if net.Driver != "" && net.Driver != "bridge" {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("network %q uses unsupported driver %q; only 'bridge' driver is supported", networkName, net.Driver),
			}
		}
	}

	return nil
}

// validateServiceQuadletCompatibility checks a single service for quadlet compatibility.
func validateServiceQuadletCompatibility(serviceName string, service types.ServiceConfig) error {
	checks := []func() error{
		func() error { return validateServiceImage(serviceName, service) },
		func() error { return validateSecuritySettings(serviceName, service) },
		func() error { return validateRestartPolicy(serviceName, service.Restart) },
		func() error { return validateDeployConfig(serviceName, service) },
		func() error { return validateNetworking(serviceName, service) },
		func() error { return validateServiceFeatures(serviceName, service) },
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}

	return nil
}

// validateServiceImage checks service image configuration.
func validateServiceImage(serviceName string, service types.ServiceConfig) error {
	if service.Image == "" && (service.Build == nil || service.Build.Dockerfile == "") {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q has no image and cannot be used with podman-systemd; use 'image' or provide a Dockerfile with explicit image", serviceName),
		}
	}
	return nil
}

// validateSecuritySettings checks security-related configurations.
func validateSecuritySettings(serviceName string, service types.ServiceConfig) error {
	for _, opt := range service.SecurityOpt {
		if !isSupportedSecurityOpt(opt) {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("service %q uses unsupported security_opt %q; supported options: label=disable, label=nested, label=type:*, label=level:*, label=filetype:*, no-new-privileges, seccomp=*, mask=*, unmask=*", serviceName, opt),
			}
		}
	}

	if service.User != "" {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses 'user' which has limited support in podman-systemd; configure user mapping through systemd directives instead", serviceName),
		}
	}

	if service.Ipc != "" && service.Ipc != "private" && service.Ipc != "shareable" {
		if isServiceNameReference(service.Ipc) {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("service %q uses IPC mode %q which references another container; podman-systemd does not support container-specific IPC sharing", serviceName, service.Ipc),
			}
		}
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses unsupported IPC mode %q; only 'private' and 'shareable' are supported", serviceName, service.Ipc),
		}
	}

	return nil
}

// validateRestartPolicy checks restart policy compatibility.
func validateRestartPolicy(serviceName string, restart string) error {
	if restart != "" && !isSupportedRestartPolicy(restart) {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses unsupported restart policy %q; only 'no', 'always', 'on-failure', and 'unless-stopped' are supported", serviceName, restart),
		}
	}
	return nil
}

// validateDeployConfig checks deployment configuration.
func validateDeployConfig(serviceName string, service types.ServiceConfig) error {
	if service.Deploy == nil {
		return nil
	}

	if service.Deploy.Replicas != nil && *service.Deploy.Replicas > 1 {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q specifies 'replicas' which is not supported by podman-systemd; systemd manages one instance per service unit", serviceName),
		}
	}

	if len(service.Deploy.Placement.Constraints) > 0 || len(service.Deploy.Placement.Preferences) > 0 {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses deploy constraints or preferences which are not supported by podman-systemd; remove placement configuration", serviceName),
		}
	}

	return nil
}

// validateNetworking checks networking configuration.
func validateNetworking(serviceName string, service types.ServiceConfig) error {
	if service.NetworkMode == "" {
		return nil
	}

	if service.NetworkMode == "none" {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses unsupported network mode %q; use 'bridge' or 'host'", serviceName, service.NetworkMode),
		}
	}

	if service.NetworkMode != "host" && service.NetworkMode != "bridge" {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses unsupported network mode %q; only 'bridge' and 'host' are supported", serviceName, service.NetworkMode),
		}
	}

	if service.NetworkMode == "host" && len(service.Ports) > 0 {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q cannot publish ports when using 'host' network mode", serviceName),
		}
	}

	return nil
}

// validateServiceFeatures checks miscellaneous service features.
func validateServiceFeatures(serviceName string, service types.ServiceConfig) error {
	// Check depends_on conditions
	for depName, condition := range service.DependsOn {
		if condition.Condition != "" && condition.Condition != "service_started" {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("service %q has unsupported depends_on condition %q on %q; only 'service_started' is supported", serviceName, condition.Condition, depName),
			}
		}
	}

	// Check logging driver
	if service.Logging != nil && service.Logging.Driver != "" {
		if service.Logging.Driver != "json-file" && service.Logging.Driver != "journald" {
			return &quadletCompatibilityError{
				message: fmt.Sprintf("service %q uses logging driver %q; only 'json-file' and 'journald' are supported by podman-systemd", serviceName, service.Logging.Driver),
			}
		}
	}

	// Check stop signal
	if service.StopSignal != "" && !isSupportedStopSignal(service.StopSignal) {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses stop signal %q; only 'SIGTERM' and 'SIGKILL' are supported by podman-systemd", serviceName, service.StopSignal),
		}
	}

	// Check tmpfs
	if len(service.Tmpfs) > 0 {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses 'tmpfs' which is not supported by podman-systemd; use named volumes or bind mounts instead", serviceName),
		}
	}

	// Check profiles
	if len(service.Profiles) > 0 {
		return &quadletCompatibilityError{
			message: fmt.Sprintf("service %q uses 'profiles' which are not supported by podman-systemd; remove profile configuration", serviceName),
		}
	}

	return nil
}

// isSupportedRestartPolicy checks if a restart policy is supported by systemd.
func isSupportedRestartPolicy(policy string) bool {
	const (
		RestartNo            = "no"
		RestartAlways        = "always"
		RestartOnFailure     = "on-failure"
		RestartUnlessStopped = "unless-stopped"
	)

	switch policy {
	case RestartNo, RestartAlways, RestartOnFailure, RestartUnlessStopped:
		return true
	default:
		return false
	}
}

// isSupportedStopSignal checks if a stop signal is supported by podman-systemd.
func isSupportedStopSignal(signal string) bool {
	// Normalize signal names (may come with or without SIG prefix)
	const (
		SigTerm = "SIGTERM"
		SigKill = "SIGKILL"
	)

	switch signal {
	case SigTerm, SigKill, "TERM", "KILL":
		return true
	default:
		return false
	}
}

// isServiceNameReference checks if an IPC/PID mode string references a service.
// Format is typically "service:name" or "container:name".
func isServiceNameReference(mode string) bool {
	if mode == "" {
		return false
	}
	// Check for service: or container: prefix
	return strings.HasPrefix(mode, "service:") || strings.HasPrefix(mode, "container:")
}

// isSupportedSecurityOpt checks if a security_opt value is supported by Quadlet.
// Supported options map to Quadlet keys: SecurityLabelDisable, SecurityLabelNested,
// SecurityLabelType, SecurityLabelLevel, SecurityLabelFileType, NoNewPrivileges,
// SeccompProfile, Mask, Unmask.
func isSupportedSecurityOpt(opt string) bool {
	switch {
	case opt == "label=disable", opt == "label:disable":
		return true
	case opt == "label=nested", opt == "label:nested":
		return true
	case strings.HasPrefix(opt, "label=type:"), strings.HasPrefix(opt, "label:type:"):
		return true
	case strings.HasPrefix(opt, "label=level:"), strings.HasPrefix(opt, "label:level:"):
		return true
	case strings.HasPrefix(opt, "label=filetype:"), strings.HasPrefix(opt, "label:filetype:"):
		return true
	case opt == "no-new-privileges", opt == "no-new-privileges:true", opt == "no-new-privileges=true":
		return true
	case strings.HasPrefix(opt, "seccomp="), strings.HasPrefix(opt, "seccomp:"):
		return true
	case strings.HasPrefix(opt, "mask="), strings.HasPrefix(opt, "mask:"):
		return true
	case strings.HasPrefix(opt, "unmask="), strings.HasPrefix(opt, "unmask:"):
		return true
	default:
		return false
	}
}

// isYAMLError checks if an error is a YAML parsing error.
func isYAMLError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains typical YAML error indicators
	return !errors.Is(err, os.ErrNotExist) && // not a file not found error
		!errors.Is(err, os.ErrPermission) // not a permission error
}

// MissingSecretsResult contains the result of checking for missing secrets.
type MissingSecretsResult struct {
	ServiceName    string
	MissingSecrets []string
}

// GetServiceSecrets returns the list of secret names used by a service.
// It reads from the x-podman-env-secrets extension.
func GetServiceSecrets(service types.ServiceConfig) []string {
	if service.Extensions == nil {
		return nil
	}

	envSecretsRaw, ok := service.Extensions["x-podman-env-secrets"]
	if !ok || envSecretsRaw == nil {
		return nil
	}

	envSecretsMap, ok := envSecretsRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	secrets := make([]string, 0, len(envSecretsMap))
	for secretName := range envSecretsMap {
		secrets = append(secrets, secretName)
	}

	return secrets
}

// GetAvailablePodmanSecrets queries podman for the list of available secrets.
func GetAvailablePodmanSecrets(ctx context.Context) (map[string]struct{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cmd := exec.CommandContext(ctx, "podman", "secret", "ls", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list podman secrets: %w", err)
	}

	secrets := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			secrets[line] = struct{}{}
		}
	}

	return secrets, nil
}

// CheckMissingSecrets checks if all secrets required by services in a project exist in podman.
// Returns a slice of MissingSecretsResult for services that have missing secrets.
func CheckMissingSecrets(ctx context.Context, project *types.Project) ([]MissingSecretsResult, error) {
	if project == nil {
		return nil, nil
	}

	availableSecrets, err := GetAvailablePodmanSecrets(ctx)
	if err != nil {
		return nil, err
	}

	var results []MissingSecretsResult

	for serviceName, service := range project.Services {
		serviceSecrets := GetServiceSecrets(service)
		if len(serviceSecrets) == 0 {
			continue
		}

		var missing []string
		for _, secret := range serviceSecrets {
			if _, exists := availableSecrets[secret]; !exists {
				missing = append(missing, secret)
			}
		}

		if len(missing) > 0 {
			results = append(results, MissingSecretsResult{
				ServiceName:    serviceName,
				MissingSecrets: missing,
			})
		}
	}

	return results, nil
}

// ServiceHasMissingSecrets checks if a specific service has any missing secrets.
// Returns the list of missing secret names.
func ServiceHasMissingSecrets(ctx context.Context, service types.ServiceConfig) ([]string, error) {
	serviceSecrets := GetServiceSecrets(service)
	if len(serviceSecrets) == 0 {
		return nil, nil
	}

	availableSecrets, err := GetAvailablePodmanSecrets(ctx)
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, secret := range serviceSecrets {
		if _, exists := availableSecrets[secret]; !exists {
			missing = append(missing, secret)
		}
	}

	return missing, nil
}

// FilterServicesWithMissingSecrets removes services with missing secrets from the project.
// It modifies project.Services in place and returns info about skipped services.
// If availableSecrets is nil, it queries podman for the list of available secrets.
func FilterServicesWithMissingSecrets(ctx context.Context, project *types.Project, availableSecrets map[string]struct{}) ([]MissingSecretsResult, error) {
	if project == nil {
		return nil, nil
	}

	var err error
	if availableSecrets == nil {
		availableSecrets, err = GetAvailablePodmanSecrets(ctx)
		if err != nil {
			return nil, err
		}
	}

	var skipped []MissingSecretsResult

	for serviceName, service := range project.Services {
		serviceSecrets := GetServiceSecrets(service)
		if len(serviceSecrets) == 0 {
			continue
		}

		var missingSecrets []string
		for _, secret := range serviceSecrets {
			if _, exists := availableSecrets[secret]; !exists {
				missingSecrets = append(missingSecrets, secret)
			}
		}

		if len(missingSecrets) > 0 {
			skipped = append(skipped, MissingSecretsResult{
				ServiceName:    serviceName,
				MissingSecrets: missingSecrets,
			})
			delete(project.Services, serviceName)
		}
	}

	return skipped, nil
}

// CheckServiceSecrets checks a single service for missing secrets against the provided available secrets map.
// Returns the list of missing secret names, or nil if all secrets are available.
func CheckServiceSecrets(service types.ServiceConfig, availableSecrets map[string]struct{}) []string {
	if availableSecrets == nil {
		return nil
	}

	serviceSecrets := GetServiceSecrets(service)
	if len(serviceSecrets) == 0 {
		return nil
	}

	var missing []string
	for _, secret := range serviceSecrets {
		if _, exists := availableSecrets[secret]; !exists {
			missing = append(missing, secret)
		}
	}

	return missing
}
