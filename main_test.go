package main

import (
	"os"
	"path"
	"testing"

	"github.com/moby/moby/pkg/stringid"
	"github.com/opencontainers/go-digest"
)

func check(t *testing.T, err error, prefix string) {
	if err != nil {
		t.Fatalf("%s, err: %s\n", prefix, err.Error())
	}
}

func checkFixedLayerNumb(t *testing.T, expected int, actual int) {
	if expected != actual {
		t.Fatalf("expected number of fixed layers != number of layers that were actually fixed; %d != %d", expected, actual)
	}
}

func checkLayerDirNotExist(t *testing.T, layerDir string) {
	_, err := os.Stat(layerDir)
	if !os.IsNotExist(err) {
		t.Fatalf("layer dir is supposed to be removed")
	}
}

func checkLayerDirsExist(t *testing.T, layerDir string, overlayDir string) {
	_, err := os.Stat(layerDir)
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, err = os.Stat(overlayDir)
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func addLayer(t *testing.T, dir string, overlayRootDir string, baseLayer bool) (string, string) {
	// make valid layer sir name which is a layer's chain ID
	layerChainID := digest.FromBytes([]byte("foobar beef meet"))

	layerDir := path.Join(dir, layerChainID.Encoded())
	err := os.MkdirAll(layerDir, 0777)
	check(t, err, "failed to create a layer dir")

	cacheID := stringid.GenerateRandomID()
	err = os.WriteFile(path.Join(layerDir, CacheIdFile), []byte(cacheID), 0644)
	check(t, err, "failed to write to cacheID file")

	err = os.WriteFile(path.Join(layerDir, DiffFile), []byte(digest.FromBytes([]byte("foo bar")).String()), 0644)
	check(t, err, "failed to write to diff file")

	err = os.WriteFile(path.Join(layerDir, SizeFile), []byte("1024"), 0644)
	check(t, err, "failed to write to size file")

	overlayDir := path.Join(overlayRootDir, cacheID)
	err = os.MkdirAll(path.Join(overlayRootDir, cacheID), 0777)
	check(t, err, "failed to create an overlay dir")

	linkID := "SNBGQO2GG7VCPWMLRKED6NNSTK" // the len must be equal to LinkIdLength
	err = os.WriteFile(path.Join(overlayRootDir, cacheID, LinkFile), []byte(linkID), 0644)
	check(t, err, "failed to write to link file")

	if !baseLayer {
		err = os.WriteFile(path.Join(layerDir, ParentFile), []byte(digest.FromBytes([]byte("foo bar parent")).String()), 0644)
		check(t, err, "failed to write to parent file")
		err = os.WriteFile(path.Join(overlayRootDir, cacheID, LowerFile), []byte("l/KFPW5XUWPXL26GW47CBVKPKAVQ"), 0644)
		check(t, err, "failed to write to lower file")
	}

	return layerDir, overlayDir
}

func TestEmptyStore(t *testing.T) {
	{
		// non existing directory
		s, err := NewDockerStore("/tmp/just-some-non-existing-path")
		check(t, err, "failed to init Docker Store")

		f, err := checkStore(s, false)
		check(t, err, "check of non-existing Docker Store dir failed")
		checkFixedLayerNumb(t, 0, f)
	}
	{
		// empty root directory
		d := t.TempDir()
		s, err := NewDockerStore(d)
		check(t, err, "failed to init Docker Store")

		f, err := checkStore(s, false)
		check(t, err, "check of empty Docker Store failed")
		checkFixedLayerNumb(t, 0, f)
	}
	{
		// empty images directory
		d := t.TempDir()
		s, err := NewDockerStore(d)
		check(t, err, "failed to init Docker Store")

		err = os.MkdirAll(s.ImagesDir(), 0777)
		check(t, err, "failed to create images dir")

		f, err := checkStore(s, false)
		check(t, err, "check of empty Images dir failed")
		checkFixedLayerNumb(t, 0, f)
	}

	{
		// empty layers directory
		d := t.TempDir()
		s, err := NewDockerStore(d)
		check(t, err, "failed to init Docker Store")

		err = os.MkdirAll(s.LayersDir(), 0777)
		check(t, err, "failed to create images dir")

		f, err := checkStore(s, false)
		check(t, err, "check of empty Layers dir failed")
		checkFixedLayerNumb(t, 0, f)
	}
}

func TestInsufficientPermission(t *testing.T) {
	// empty layers directory
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	err = os.MkdirAll(path.Join(s.Root(), "image"), 0777)
	check(t, err, "failed to create image dir")

	// make it impossible to read/enter directory
	err = os.Mkdir(s.ImagesDir(), os.ModeDir)
	check(t, err, "failed to create image dir")

	f, err := checkStore(s, false)
	if err == nil {
		t.Fatalf("the store checking should fail if there is insufficient permissions")
	}
	checkFixedLayerNumb(t, -1, f)
}

func TestInvalidLayerDirName(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	// invalid dir name, it has to be sha256 digest (chain ID)
	layerDir := path.Join(s.LayersDir(), "foobar")
	err = os.MkdirAll(layerDir, 0777)
	check(t, err, "failed to create a layer dir")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestEmptyLayerDir(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	// make valid layer sir name which is a layer's chain ID
	layerChainID := digest.FromBytes([]byte("foobar beef meet"))

	layerDir := path.Join(s.LayersDir(), layerChainID.Encoded())
	err = os.MkdirAll(layerDir, 0777)
	check(t, err, "failed to create a layer dir")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestInvalidCacheID(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	// make valid layer structure
	layerDir, _ := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make cacheID file invalid
	err = os.WriteFile(path.Join(layerDir, CacheIdFile), []byte("foobar"), 0644)
	check(t, err, "failed to write to cacheID file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestInvalidDiffID(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, _ := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make diffID file invalid
	err = os.WriteFile(path.Join(layerDir, DiffFile), []byte("sha256:foobar123"), 0644)
	check(t, err, "failed to write to diff file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestInvalidSizeFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, _ := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make size file invalid
	err = os.WriteFile(path.Join(layerDir, SizeFile), []byte("not number"), 0644)
	check(t, err, "failed to write to diff file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestMissingOverlayDir(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// remove overlay dir
	os.RemoveAll(overlayDir)

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestMissingLinkFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// remove link file
	os.Remove(path.Join(overlayDir, LinkFile))

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestEmptyLinkFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make link file empty
	err = os.WriteFile(path.Join(overlayDir, LinkFile), []byte(""), 0644)
	check(t, err, "failed to write to a link file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestInvalidParentFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, _ := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make parent file invalid
	err = os.WriteFile(path.Join(layerDir, ParentFile), []byte("sha256:2377HHHHH"), 0644)
	check(t, err, "failed to write to a link file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestMissingLowerFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// remove lower file
	os.Remove(path.Join(overlayDir, LowerFile))

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestEmptyLowerFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make lower file empty
	err = os.WriteFile(path.Join(overlayDir, LowerFile), []byte(""), 0644)
	check(t, err, "failed to write to a lower file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
}

func TestIncorrectLowerFile(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	imageDbDir := path.Join(s.ImagesDir(), "imagedb")
	err = os.MkdirAll(imageDbDir, 0777)
	check(t, err, "failed to create an image DB dir")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	// make lower file invalid
	err = os.WriteFile(path.Join(overlayDir, LowerFile), []byte("l/GFFFD"), 0644)
	check(t, err, "failed to write to a lower file")

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirNotExist(t, layerDir)
	checkFixedLayerNumb(t, 1, f)
	_, err = os.Stat(imageDbDir)
	if !os.IsNotExist(err) {
		t.Fatalf("an image DB/metadata dir is supposed to be removed if at leats one layer is fixed/removed")
	}
}

func TestCorrectLayer(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), false)

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirsExist(t, layerDir, overlayDir)
	checkFixedLayerNumb(t, 0, f)
}

func TestCorrectBaseLayer(t *testing.T) {
	d := t.TempDir()
	s, err := NewDockerStore(d)
	check(t, err, "failed to init Docker Store")

	layerDir, overlayDir := addLayer(t, s.LayersDir(), s.GraphDriverDir(), true)

	f, err := checkStore(s, true)
	check(t, err, "check of invalid layer dir failed")
	checkLayerDirsExist(t, layerDir, overlayDir)
	checkFixedLayerNumb(t, 0, f)
}
