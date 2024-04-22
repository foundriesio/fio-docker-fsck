package main

import (
	_ "crypto/sha256" // must be imported before go-digest
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/moby/moby/pkg/stringid"
	"github.com/opencontainers/go-digest"
)

type (
	Layer struct {
		Dir     string
		ChainID digest.Digest
		CacheID string
		Diff    digest.Digest
		Size    uint64
		Parent  *digest.Digest
		Overlay Overlay
	}

	Overlay struct {
		Dir   string
		Link  string
		Lower string
	}
)

const (
	CacheIdFile = "cache-id"
	DiffFile    = "diff"
	ParentFile  = "parent"
	SizeFile    = "size"

	LinkFile  = "link"
	LowerFile = "lower"

	LinkIdLength = 26
)

func parseLayer(dir string, chainID string, overlayRootDir string) (Layer, error) {
	var err error
	l := Layer{Dir: dir}

	// check chain ID (layer directory name)
	if l.ChainID, err = digest.Parse(fmt.Sprintf("%s:%s", digest.SHA256, chainID)); err != nil {
		return l, err
	}

	// read and check cache ID, stringid.GenerateRandomID(), see https://github.com/moby/moby/blob/6f6b9d2e67a8867672ff4eb35e8907af14b1bba3/layer/layer_store.go#L316
	if b, err := os.ReadFile(path.Join(dir, CacheIdFile)); err == nil {
		l.CacheID = string(b)
	} else {
		return l, err
	}
	if err := stringid.ValidateID(l.CacheID); err != nil {
		return l, err
	}

	// read and parse diff ID
	if l.Diff, err = newDigestFromFile(path.Join(dir, DiffFile)); err != nil {
		return l, err
	}

	if b, err := os.ReadFile(path.Join(dir, SizeFile)); err == nil {
		if l.Size, err = strconv.ParseUint(string(b), 0, 0); err != nil {
			return l, err
		}
	} else {
		return l, err
	}

	// read and parse parent ID if exists
	_, err = os.Stat(path.Join(dir, ParentFile))

	if err == nil {
		parent, err := newDigestFromFile(path.Join(dir, ParentFile))
		if err != nil {
			return l, err
		}
		l.Parent = &parent
	} else if !os.IsNotExist(err) {
		return l, err
	}

	if l.Overlay, err = newOverlay(path.Join(overlayRootDir, l.CacheID), l.Parent == nil); err != nil {
		return l, err
	}

	return l, nil
}

func (l Layer) Remove() error {
	if l.Dir == "" {
		return nil
	}
	err := os.RemoveAll(l.Dir)
	if err != nil {
		return err
	}
	if l.Overlay.Dir == "" {
		return nil
	}
	err = os.RemoveAll(l.Overlay.Dir)
	if err != nil {
		return err
	}
	return nil
}

func newDigestFromFile(f string) (digest.Digest, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return "", err
	}
	return digest.Parse(string(b))
}

func newOverlay(dir string, baseLayer bool) (Overlay, error) {
	o := Overlay{Dir: dir}

	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Overlay{}, fmt.Errorf("layer's overlay dir doesn't exist: %s", dir)
		}
		return Overlay{}, err
	}

	b, err := os.ReadFile(path.Join(dir, LinkFile))
	if err != nil {
		return o, err
	} else {
		o.Link = string(b)
	}

	if len(o.Link) != LinkIdLength {
		return o, fmt.Errorf("a link ID file contains an invalid content/ID: %s", path.Join(dir, LinkFile))
	}

	if !baseLayer {
		b, err = os.ReadFile(path.Join(dir, LowerFile))
		if err != nil {
			return o, err
		} else {
			o.Lower = string(b)
		}

		// lower file contains a list of underlying layer link IDs, so it must have at list one link ID
		if len(o.Lower) < LinkIdLength {
			return o, fmt.Errorf("incorrect lower file: %s", path.Join(dir, LowerFile))
		}
	}
	return o, nil
}
