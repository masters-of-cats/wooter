package wooter_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/julz/wooter"
)

var testLogger = lagertest.NewTestLogger("w00t")

func TestWootASingleLayer(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	dir, err := ioutil.TempDir("", "woot")
	if err != nil {
		t.Fatal("tmpdir", err)
	}

	w := wooter.Cp{
		BaseDir: dir,
	}

	if err := w.Unpack(testLogger, "my-layer-id", "", mytar); err != nil {
		t.Errorf("expected unpack to succeed but got error %s", err)
	}

	bundle, err := w.Bundle(testLogger, "my-container-id", []string{"my-layer-id"})
	if err != nil {
		t.Errorf("expected creating bundle to succeed but got error %s", err)
	}

	if _, err := os.Stat(filepath.Join(bundle.Root.Path)); err != nil {
		t.Errorf("expected root path to inside the returned bundle to exist")
	}

	if _, err = os.Stat(filepath.Join(bundle.Root.Path, "foo", "bar")); err != nil {
		t.Errorf("expected foo/bar to exist inside the returned root")
	}
}

func TestExistingAWoot(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	dir, err := ioutil.TempDir("", "woot")
	if err != nil {
		t.Fatal("tmpdir", err)
	}

	w := wooter.Cp{
		BaseDir: dir,
	}

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

func TestWootingWithAParentWoot(t *testing.T) {
	mytar, err := os.Open("mytar.tar")
	if err != nil {
		t.Fatal("open mytar", err)
	}

	myparenttar, err := os.Open("myparent.tar")
	if err != nil {
		t.Fatal("open parent tar", err)
	}

	dir, err := ioutil.TempDir("", "woot")
	if err != nil {
		t.Fatal("tmpdir", err)
	}

	w := wooter.Cp{
		BaseDir: dir,
	}

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
