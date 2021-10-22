package main

import (
	"io/ioutil"
	"os"
	"path"
)

type (
	DockerStore interface {
		Root() string
		ImagesDir() string
		LayersDir() string
		GraphDriverDir() string
		ReadLayersDir() ([]os.FileInfo, error)
		ParseLayerDir(dirInfo os.FileInfo) (Layer, error)
	}

	dockerStore struct {
		root        string
		graphDriver string
		imagesDir   string
		layersDir   string
		graphDir    string
	}
)

const (
	DefaultGraphDriver = "overlay2"
)

func NewDockerStore(root string) (DockerStore, error) {
	graphDriver := DefaultGraphDriver
	i := path.Join(root, "image", graphDriver)
	l := path.Join(i, "layerdb/sha256")
	g := path.Join(root, graphDriver)

	return &dockerStore{root: root, graphDriver: graphDriver, imagesDir: i, layersDir: l, graphDir: g}, nil
}

func (d *dockerStore) Root() string {
	return d.root
}

func (d *dockerStore) ImagesDir() string {
	return d.imagesDir
}

func (d *dockerStore) LayersDir() string {
	return d.layersDir
}

func (d *dockerStore) GraphDriverDir() string {
	return d.graphDir
}

func (d *dockerStore) ReadLayersDir() ([]os.FileInfo, error) {
	_, err := os.Stat(d.layersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []os.FileInfo{}, nil
		}
		return nil, err
	}

	return ioutil.ReadDir(d.layersDir)
}

func (d *dockerStore) ParseLayerDir(dirInfo os.FileInfo) (Layer, error) {
	return parseLayer(path.Join(d.layersDir, dirInfo.Name()), dirInfo.Name(), d.graphDir)
}
