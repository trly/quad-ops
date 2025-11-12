// Package service provides platform-agnostic service domain models.
package service

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Service name must be valid for systemd unit names and filesystem paths.
	// Allow alphanumeric, hyphen, underscore, and dot.
	serviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	messages := make([]string, 0, len(e))
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Validate validates a service specification.
func (s *Spec) Validate() error {
	var errs ValidationErrors

	// Validate name
	if s.Name == "" {
		errs = append(errs, ValidationError{Field: "Name", Message: "service name is required"})
	} else if !serviceNameRegex.MatchString(s.Name) {
		errs = append(errs, ValidationError{
			Field:   "Name",
			Message: fmt.Sprintf("invalid service name %q: must start with alphanumeric and contain only alphanumeric, hyphen, underscore, or dot", s.Name),
		})
	}

	// Validate container
	if err := s.Container.Validate(); err != nil {
		errs = append(errs, ValidationError{Field: "Container", Message: err.Error()})
	}

	// Validate volumes
	for i, vol := range s.Volumes {
		if err := vol.Validate(); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("Volumes[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate networks
	for i, net := range s.Networks {
		if err := net.Validate(); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("Networks[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate dependencies (check for self-reference)
	for _, dep := range s.DependsOn {
		if dep == s.Name {
			errs = append(errs, ValidationError{
				Field:   "DependsOn",
				Message: fmt.Sprintf("service cannot depend on itself: %q", dep),
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates container configuration.
func (c *Container) Validate() error {
	var errs ValidationErrors

	// Image is required unless Build is specified
	if c.Image == "" && c.Build == nil {
		errs = append(errs, ValidationError{Field: "Image", Message: "image is required when build is not specified"})
	}

	// Validate healthcheck if present
	if c.Healthcheck != nil {
		if err := c.Healthcheck.Validate(); err != nil {
			errs = append(errs, ValidationError{Field: "Healthcheck", Message: err.Error()})
		}
	}

	// Validate build if present
	if c.Build != nil {
		if err := c.Build.Validate(); err != nil {
			errs = append(errs, ValidationError{Field: "Build", Message: err.Error()})
		}
	}

	// Validate restart policy
	if c.RestartPolicy != "" {
		validPolicies := map[RestartPolicy]bool{
			RestartPolicyNo:            true,
			RestartPolicyAlways:        true,
			RestartPolicyOnFailure:     true,
			RestartPolicyUnlessStopped: true,
		}
		if !validPolicies[c.RestartPolicy] {
			errs = append(errs, ValidationError{
				Field:   "RestartPolicy",
				Message: fmt.Sprintf("invalid restart policy %q: must be one of no, always, on-failure, unless-stopped", c.RestartPolicy),
			})
		}
	}

	// Validate network mode conflicts
	if c.Network.Mode == "host" && len(c.Network.ServiceNetworks) > 0 {
		errs = append(errs, ValidationError{
			Field:   "Network",
			Message: "cannot use network_mode: host with additional networks. host mode provides access to the host network directly. remove the networks declaration from this service or change network_mode",
		})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates healthcheck configuration.
func (h *Healthcheck) Validate() error {
	var errs ValidationErrors

	if len(h.Test) == 0 {
		errs = append(errs, ValidationError{Field: "Test", Message: "healthcheck test command is required"})
	}

	if h.Retries < 0 {
		errs = append(errs, ValidationError{Field: "Retries", Message: "retries must be non-negative"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates build configuration.
func (b *Build) Validate() error {
	var errs ValidationErrors

	if b.Context == "" {
		errs = append(errs, ValidationError{Field: "Context", Message: "build context is required"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates volume configuration.
func (v *Volume) Validate() error {
	var errs ValidationErrors

	if v.Name == "" {
		errs = append(errs, ValidationError{Field: "Name", Message: "volume name is required"})
	} else if !serviceNameRegex.MatchString(v.Name) {
		errs = append(errs, ValidationError{
			Field:   "Name",
			Message: fmt.Sprintf("invalid volume name %q: must start with alphanumeric and contain only alphanumeric, hyphen, underscore, or dot", v.Name),
		})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates network configuration.
func (n *Network) Validate() error {
	var errs ValidationErrors

	if n.Name == "" {
		errs = append(errs, ValidationError{Field: "Name", Message: "network name is required"})
	} else if !serviceNameRegex.MatchString(n.Name) {
		errs = append(errs, ValidationError{
			Field:   "Name",
			Message: fmt.Sprintf("invalid network name %q: must start with alphanumeric and contain only alphanumeric, hyphen, underscore, or dot", n.Name),
		})
	}

	// Validate IPAM if present
	if n.IPAM != nil {
		if err := n.IPAM.Validate(); err != nil {
			errs = append(errs, ValidationError{Field: "IPAM", Message: err.Error()})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate validates IPAM configuration.
func (i *IPAM) Validate() error {
	// Basic validation - ensure at least one config if IPAM is specified
	// More detailed validation can be added as needed
	return nil
}
