package wooter_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/julz/wooter"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const Maximus = uint32(wooter.Maximus)

var testLogger = lagertest.NewTestLogger("w00t")

func TestWootASingleLayer(t *testing.T) {
	bundle := createSingleLayerBundle(t, createWoot(t, false))

	if _, err := os.Stat(filepath.Join(bundle.Root.Path)); err != nil {
		t.Errorf("expected root path to inside the returned bundle to exist")
	}

	if _, err := os.Stat(filepath.Join(bundle.Root.Path, "foo", "bar")); err != nil {
		t.Errorf("expected foo/bar to exist inside the returned root")
	}
}

func TestExistingAWoot(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	w := createWoot(t, false)

	if w.Exists(testLogger, "my-layer-id") {
		t.Error("expected my-id not to exist before unpacking")
	}

	if err := w.Unpack(testLogger, "my-layer-id", "", mytar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	if w.Exists(testLogger, "my-layer-id") != true {
		t.Error("expected my-id to exist after unpacking")
	}
}

func TestDeletingAWoot(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	w := createWoot(t, false)

	if err := w.Unpack(testLogger, "my-layer-id", "", mytar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	if err := w.Delete(testLogger, "my-layer-id"); err != nil {
		t.Errorf("expected delete to succeed but got error %s", err)
	}

	if w.Exists(testLogger, "my-layer-id") {
		t.Error("expected my-id not to exist after deleting")
	}
}

func TestWootingWithAParentWoot(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	myparenttar, err := os.Open("myparent.tar")
	if err != nil {
		t.Fatal("open parent tar", err)
	}

	w := createWoot(t, false)

	if err := w.Unpack(testLogger, "my-parent-layer-id", "", myparenttar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	if err := w.Unpack(testLogger, "my-layer-id", "my-parent-layer-id", mytar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	bundle, err := w.Bundle(testLogger, "my-container-id", []string{"my-parent-layer-id", "my-layer-id"})
	if err != nil {
		t.Errorf("expected creating bundle to succeed but got error %s", err)
	}

	if _, err = os.Stat(filepath.Join(bundle.Root.Path)); err != nil {
		t.Errorf("expected root path to inside the returned bundle to exist")
	}

	if _, err = os.Stat(filepath.Join(bundle.Root.Path, "foo", "bar")); err != nil {
		t.Errorf("expected foo/bar to exist inside the returned root")
	}

	if _, err = os.Stat(filepath.Join(bundle.Root.Path, "i", "am", "parent")); err != nil {
		t.Errorf("expected i/am/parent to exist inside the returned root")
	}
}

func TestChownsToMaximusIfNotPrivileged(t *testing.T) {
	bundle := createSingleLayerBundle(t, createWoot(t, false))

	stat, err := os.Stat(filepath.Join(bundle.Root.Path, "foo"))
	if err != nil {
		t.Errorf("expected foo to exist inside the returned root")
	}

	uid := stat.Sys().(*syscall.Stat_t).Uid
	gid := stat.Sys().(*syscall.Stat_t).Gid
	if uid != Maximus || gid != Maximus {
		t.Errorf("expected foo to be owned by Maximus")
	}

}

func TestNoChownIfPrivileged(t *testing.T) {
	bundle := createSingleLayerBundle(t, createWoot(t, true))

	stat, err := os.Stat(filepath.Join(bundle.Root.Path, "foo"))
	if err != nil {
		t.Errorf("expected foo to exist inside the returned root")
	}

	uid := stat.Sys().(*syscall.Stat_t).Uid
	gid := stat.Sys().(*syscall.Stat_t).Gid

	if uid == Maximus || gid == Maximus {
		t.Errorf("expected foo not to be owned by Maximus")
	}
}

func createSingleLayerBundle(t *testing.T, driver groot.Driver) specs.Spec {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	if err := driver.Unpack(testLogger, "my-layer-id", "", mytar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	bundle, err := driver.Bundle(testLogger, "my-container-id", []string{"my-layer-id"})
	if err != nil {
		t.Errorf("expected creating bundle to succeed but got error %s", err)
	}

	return bundle
}

func createWoot(t *testing.T, privileged bool) wooter.Cp {
	dir, err := ioutil.TempDir("", "woot")
	if err != nil {
		t.Fatal("tmpdir", err)
	}

	return wooter.Cp{
		BaseDir:    dir,
		Privileged: privileged,
	}
}
