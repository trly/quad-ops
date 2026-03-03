package systemd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/state"
)

// ComputeUnitState computes content and bind mount hashes for change detection.
// It hashes the rendered unit file content and any bind-mounted regular files
// whose source paths are within the project directory.
func ComputeUnitState(unit Unit, project *types.Project, repoPath string) state.UnitState {
	var buf bytes.Buffer
	_, _ = unit.File.WriteTo(&buf)
	contentHash := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))

	bindMountHashes := CollectBindMountHashes(project, repoPath)

	return state.UnitState{
		ContentHash:     contentHash,
		BindMountHashes: bindMountHashes,
	}
}

// CollectBindMountHashes computes SHA256 hashes for bind-mounted regular files
// within the project directory. Files outside the project dir, directories,
// and unreadable files are skipped.
func CollectBindMountHashes(project *types.Project, repoPath string) map[string]string {
	hashes := make(map[string]string)
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return hashes
	}

	for _, svc := range project.Services {
		for _, vol := range svc.Volumes {
			if vol.Type != types.VolumeTypeBind || vol.Source == "" {
				continue
			}

			source := vol.Source
			if !filepath.IsAbs(source) {
				source = filepath.Join(project.WorkingDir, source)
			}

			absSource, err := filepath.Abs(source)
			if err != nil {
				continue
			}

			// Only hash files within the project directory
			if !strings.HasPrefix(absSource, absRepoPath+string(filepath.Separator)) {
				continue
			}

			info, err := os.Stat(absSource)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}

			data, err := os.ReadFile(absSource)
			if err != nil {
				continue
			}

			hashes[absSource] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	}

	return hashes
}
