package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var dataRoot string
	var fixStore bool

	flag.StringVar(&dataRoot, "data-root", "/var/lib/docker", "A path to docker data root")
	flag.BoolVar(&fixStore, "fix-store", false, "A flag to turn ON store fixing (removes broken layers)")
	flag.Parse()

	s, err := NewDockerStore(dataRoot)
	if err != nil {
		log.Fatalf("failed to initialize Docker Store: %s", err.Error())
	}
	fn, err := checkStore(s, fixStore)
	if err != nil {
		log.Fatalf("failed to check Docker Store: %s", err.Error())
		os.Exit(1)
	} else if fixStore {
		log.Printf("fixed %d layers", fn)
	}
}

func checkStore(s DockerStore, fixStore bool) (int, error) {
	layerDirs, err := s.ReadLayersDir()
	if err != nil {
		return -1, err
	}

	layersToRemove := make(map[string]Layer)
	for _, d := range layerDirs {
		l, err := s.ParseLayerDir(d)
		if err != nil {
			log.Printf("layer parse error; dir: %s, err: %s", l.Dir, err)
			layersToRemove[l.ChainID.Encoded()] = l
			// no point in further Layer verification if its parsing failed
			continue
		}

		// TODO: Add layer consistency checking, for example
		// 1) Check if chain of layers is correct for each image
		// 2) Check if each layer overlay content is mountable, the overlay driver can mount it
		// 3) Add parsing and verification of the image and distribution part of <data-root>/image/overlay2/ (imagedb, distribution)
		//log.Printf("Checking layer consistency: %s", l.Dir)
	}

	log.Printf("found %d broken layers\n", len(layersToRemove))

	if fixStore {
		var err error
		if len(layersToRemove) > 0 {
			err := s.RemoveImageMetadata()
			if err != nil {
				return -1, fmt.Errorf("failed to remove metadata of images: %s", err.Error())
			}
		}
		for _, l := range layersToRemove {
			log.Printf("removing layer: %s", l.Dir)
			if e := l.Remove(); e != nil {
				log.Printf("failed to remove layer: %s; err: %s", l.Dir, e.Error())
				err = e
			}
		}
		if err != nil {
			return -1, fmt.Errorf("failed to remove broken layers: %s", err.Error())
		}
	} else if len(layersToRemove) > 0 {
		log.Printf("skip broken layers removal")
	}

	return len(layersToRemove), nil
}
