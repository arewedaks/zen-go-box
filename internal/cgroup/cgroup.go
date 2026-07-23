package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/arewedaks/zengobox/internal/config"
)

// Apply Cgroup membatasi penggunaan resource CPU, memori, & disk IO untuk PID proxy core
func Apply(pid int, cfg *config.Config) error {
	pidStr := strconv.Itoa(pid)

	// 1. MemCG (Memory limit)
	if cfg.Cgroup.MemCG.Enabled {
		memcgDir := "/sys/fs/cgroup/memory/zengobox"
		_ = os.MkdirAll(memcgDir, 0755)

		// Set limit memory
		limitPath := filepath.Join(memcgDir, "memory.limit_in_bytes")
		_ = os.WriteFile(limitPath, []byte(cfg.Cgroup.MemCG.Limit), 0644)

		// Masukkan PID ke cgroup memory task
		tasksPath := filepath.Join(memcgDir, "tasks")
		if err := os.WriteFile(tasksPath, []byte(pidStr), 0644); err != nil {
			return fmt.Errorf("failed to apply memcg: %w", err)
		}
	}

	// 2. CPUSet (CPU Pinning)
	if cfg.Cgroup.CPUSet.Enabled {
		cpusetDir := "/sys/fs/cgroup/cpuset/zengobox"
		_ = os.MkdirAll(cpusetDir, 0755)

		// Masukkan PID ke cgroup cpuset task
		tasksPath := filepath.Join(cpusetDir, "tasks")
		if err := os.WriteFile(tasksPath, []byte(pidStr), 0644); err != nil {
			return fmt.Errorf("failed to apply cpuset: %w", err)
		}
	}

	// 3. BlkIO (IO limiting)
	if cfg.Cgroup.BlkIO.Enabled {
		blkioDir := "/sys/fs/cgroup/blkio/zengobox"
		_ = os.MkdirAll(blkioDir, 0755)

		// Masukkan PID ke blkio tasks
		tasksPath := filepath.Join(blkioDir, "tasks")
		if err := os.WriteFile(tasksPath, []byte(pidStr), 0644); err != nil {
			return fmt.Errorf("failed to apply blkio: %w", err)
		}
	}

	return nil
}
