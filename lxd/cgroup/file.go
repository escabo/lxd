package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/canonical/lxd/shared"
)

// NewFileReadWriter returns a CGroup instance using the filesystem as its backend.
func NewFileReadWriter(pid int, unifiedCapable bool) (*CGroup, error) {
	// Setup the read/writer struct.
	rw := fileReadWriter{}

	// Locate the base path for each controller.
	rw.paths = map[string]string{}

	controllers, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return nil, err
	}

	for line := range strings.SplitSeq(string(controllers), "\n") {
		// Skip empty lines.
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract the fields.
		fields := strings.Split(line, ":")

		// Determine the mount path.
		path := filepath.Join("/sys/fs/cgroup", fields[1], fields[2])
		if fields[0] == "0" {
			fields[1] = "unified"
			if shared.PathExists("/sys/fs/cgroup/unified") {
				path = filepath.Join("/sys/fs/cgroup", "unified", fields[2])
			} else {
				path = filepath.Join("/sys/fs/cgroup", fields[2])
			}

			if strings.HasSuffix(fields[2], "/init.scope") {
				path = filepath.Dir(path)
			}
		}

		// Add the controllers individually.
		for ctrl := range strings.SplitSeq(fields[1], ",") {
			rw.paths[ctrl] = path
		}
	}

	cg, err := New(&rw)
	if err != nil {
		return nil, err
	}

	cg.UnifiedCapable = unifiedCapable
	return cg, nil
}

type fileReadWriter struct {
	paths map[string]string
}

// Get returns the value of a cgroup key for a specific controller.
func (rw *fileReadWriter) Get(version Backend, controller string, key string) (string, error) {
	path := filepath.Join(rw.paths[controller], key)
	if cgLayout == CgroupsUnified {
		path = filepath.Join(rw.paths["unified"], key)
	}

	value, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

// Set applies the given value to a cgroup key for a specific controller.
func (rw *fileReadWriter) Set(version Backend, controller string, key string, value string) error {
	path := filepath.Join(rw.paths[controller], key)
	if cgLayout == CgroupsUnified {
		path = filepath.Join(rw.paths["unified"], key)
	}

	return os.WriteFile(path, []byte(value), 0600)
}
