//go:build linux && cgo
// +build linux,cgo

package cmds

import (
	"os"

	"github.com/containerd/containerd/pkg/userns"
	"github.com/pkg/errors"
	"github.com/rootless-containers/rootlesskit/pkg/parent/cgrouputil"
)

func EvacuateCgroup2() error {
	if os.Getpid() == 1 && !userns.RunningInUserNS() {
		if err := cgrouputil.EvacuateCgroup2("init"); err != nil {
			return errors.Wrap(err, "failed to evacuate root cgroup")
		}
	}
	return nil
}
