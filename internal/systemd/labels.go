package systemd

import (
	"fmt"

	"github.com/trly/quad-ops/internal/buildinfo"
)

const labelPrefix = "com.github.trly.quad-ops"

// RepositoryMeta holds repository metadata used to generate fixed labels
// on all Quadlet unit types (containers, volumes, networks).
type RepositoryMeta struct {
	Name       string
	URL        string
	Ref        string
	ComposeDir string
}

// baseLabels returns the fixed set of quad-ops labels that are always applied
// to generated units as Label=key=value shadow entries.
func baseLabels(repo RepositoryMeta) []string {
	labels := []string{
		fmt.Sprintf("%s.version=%s", labelPrefix, buildinfo.Version),
		fmt.Sprintf("%s.repository.name=%s", labelPrefix, repo.Name),
		fmt.Sprintf("%s.repository.url=%s", labelPrefix, repo.URL),
	}
	if repo.Ref != "" {
		labels = append(labels, fmt.Sprintf("%s.repository.ref=%s", labelPrefix, repo.Ref))
	}
	if repo.ComposeDir != "" {
		labels = append(labels, fmt.Sprintf("%s.repository.compose-dir=%s", labelPrefix, repo.ComposeDir))
	}
	return labels
}

// applyBaseLabels merges the fixed quad-ops labels into a shadow map.
func applyBaseLabels(shadows map[string][]string, repo RepositoryMeta) {
	shadows["Label"] = append(shadows["Label"], baseLabels(repo)...)
}
