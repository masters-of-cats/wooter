package wooter

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const VolumesDir string = "volumes"
const DiffsDir string = "diffs"
const Maximus int = 4294967294

type Cp struct {
	BaseDir    string
	Privileged bool
}

func (c Cp) Unpack(logger lager.Logger, id, parentID string, tar io.Reader) error {
	logger = logger.Session("unpack")
	dest := filepath.Join(c.BaseDir, VolumesDir, id)

	logger.Info("creating-dir", lager.Data{
		"dir": dest,
	})

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	parentDir := filepath.Join(c.BaseDir, VolumesDir, parentID)
	if parentID != "" && !isEmptyDir(parentDir) {
		command := fmt.Sprintf("cp -R -a %s/. %s", filepath.Join(c.BaseDir, VolumesDir, parentID), dest+"/")
		logger.Info("copy-parent-layer-command", lager.Data{
			"command": command,
		})
		cpCmd := exec.Command("sh", "-c", command)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s", string(out), err)
		}
	}

	logger.Info("untar-layer-command", lager.Data{
		"command": fmt.Sprintf("tar -x -C %s", dest),
	})

	tarCmd := exec.Command("tar", "-p", "-x", "-C", dest)
	tarCmd.Stdin = tar
	if err := tarCmd.Run(); err != nil {
		return err
	}

	return nil
}

func (c Cp) Bundle(logger lager.Logger, handle string, layerIds []string) (specs.Spec, error) {
	logger = logger.Session("bundle")
	volumeDir := filepath.Join(c.BaseDir, VolumesDir, layerIds[len(layerIds)-1])
	destDir := filepath.Join(c.BaseDir, DiffsDir, handle)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return specs.Spec{}, err
	}

	if !isEmptyDir(volumeDir) {
		command := fmt.Sprintf("cp -R -a %s/. %s", volumeDir, destDir+"/")
		logger.Info("copy-rootfs-layer-command", lager.Data{
			"command": command,
		})

		cpCmd := exec.Command("sh", "-c", command)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			return specs.Spec{}, fmt.Errorf("%s: %s", string(out), err)
		}

		if !c.Privileged {
			err := chownToMaximus(destDir, logger)
			if err != nil {
				return specs.Spec{}, err
			}
		}
	}

	return specs.Spec{
		Root: &specs.Root{
			Path: destDir,
		},
	}, nil
}

func (c Cp) Exists(logger lager.Logger, id string) bool {
	_, err := os.Stat(filepath.Join(c.BaseDir, VolumesDir, id))
	return err == nil
}

func chownToMaximus(path string, logger lager.Logger) error {
	return recursiveChown(path, Maximus, Maximus, logger)
}

func recursiveChown(path string, uid, gid int, logger lager.Logger) error {
	logger = logger.Session("recursive-chown")

	logger.Info("recursive-chown-start", lager.Data{
		"path": path,
	})
	defer logger.Info("recursive-chown-end")
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if isSymlink(info) {
			// Do not chown symlinks, we'll be eventually chowning the files they link to instead
			return nil
		}

		return os.Chown(name, uid, gid)
	})
}

func isEmptyDir(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return true
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err == io.EOF
}

func isSymlink(info os.FileInfo) bool {
	return (info.Mode() & os.ModeSymlink) != 0
}
