package netfilter

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type IPT struct {
	binary string // "iptables" atau "ip6tables"
	hasWait bool   // true jika binary support -w flag
}

func NewIPT(binary string) *IPT {
	ipt := &IPT{binary: binary}
	ipt.detectFeatures()
	return ipt
}

func (i *IPT) detectFeatures() {
	// Cek apakah command support -w flag (wait lock)
	cmd := exec.Command(i.binary, "-w", "-L", "-n")
	err := cmd.Run()
	if err == nil {
		i.hasWait = true
	}
}

// Exec menjalankan command iptables/ip6tables dengan argumen yang diberikan
func (i *IPT) Exec(args ...string) error {
	var finalArgs []string
	if i.hasWait {
		finalArgs = append(finalArgs, "-w", "100")
	}
	finalArgs = append(finalArgs, args...)

	var stderr bytes.Buffer
	cmd := exec.Command(i.binary, finalArgs...)
	cmd.Stderr = &stderr

	slog.Debug("Executing netfilter command", "bin", i.binary, "args", strings.Join(finalArgs, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s %s: %w (stderr: %s)", i.binary, strings.Join(finalArgs, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// ExecIgnoreError menjalankan command iptables tapi mengabaikan jika ada error (berguna saat cleanup)
func (i *IPT) ExecIgnoreError(args ...string) {
	_ = i.Exec(args...)
}
