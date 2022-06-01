package main

import (
	"fmt"
	"log"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func append_multi_arch(path string, image v1.Image, platform *v1.Platform) (v1.ImageIndex, error) {

	p, err := layout.FromPath(path)
	var top v1.ImageIndex
	if err == nil {
		top, err = p.ImageIndex() // index.json (top-level index)
		if err != nil {
			return nil, err
		}
		m, err := top.IndexManifest()
		if err != nil {
			return nil, err
		}
		if len(m.Manifests) != 1 {
			return nil, fmt.Errorf("len(manifests) != 1: %d", len(m.Manifests))
		}
		v := m.Manifests[0]
		// get nested index
		index, err := top.ImageIndex(v.Digest)
		if err != nil {
			return nil, err
		}
		mt, err := index.MediaType()
		if err != nil {
			return nil, err
		}
		if mt != types.OCIImageIndex {
			return nil, fmt.Errorf("not OCI image-index: mt: %v, digest: %v", mt, v.Digest)
		}
		top = mutate.RemoveManifests(top, match.Digests(v.Digest))
		m, err = top.IndexManifest()
		if err != nil {
			return nil, err
		}
		if len(m.Manifests) != 0 {
			return nil, fmt.Errorf("len(manifests) != 0: %d", len(m.Manifests))
		}
		err = p.RemoveBlob(v.Digest)
		if err != nil {
			return nil, err
		}
		// then remove the top level
		top = index
	} else {
		top = empty.Index
		p = layout.Path(path)
	}

	// copy blobs of the single arch image to the multi arch path
	err = p.WriteImage(image)
	if err != nil {
		return nil, err
	}

	index := mutate.AppendManifests(top, mutate.IndexAddendum{
		Add: image,
		Descriptor: v1.Descriptor{
			Platform: platform,
		},
	})

	top = mutate.AppendManifests(empty.Index, mutate.IndexAddendum{Add: index})
	_, err = layout.Write(path, top)
	if err != nil {
		return nil, err
	}

	return top, nil
}

func load_single_arch(path string) (v1.Image, error) {
	p, err := layout.FromPath(path)
	if err != nil {
		return nil, err
	}
	top, err := p.ImageIndex() // ImageIndex
	if err != nil {
		return nil, err
	}
	m, err := top.IndexManifest() // IndexManifest (OCI image index)
	if err != nil {
		return nil, err
	}
	if len(m.Manifests) != 1 { // this must be a single arch image
		return nil, fmt.Errorf("len(manifests) != 1")
	}
	v := m.Manifests[0]

	image, err := top.Image(v.Digest) // v1.Image
	if err != nil {
		return nil, err
	}
	mt, err := image.MediaType()
	if err != nil {
		return nil, err
	}
	if mt != types.OCIManifestSchema1 {
		return nil, fmt.Errorf("%s image media type is not OCI manifest-schema v1: %v", path, mt)

	}
	return image, nil
}

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("usage: %s MULTIARCH_IMAGE SINGLEARCH_IMAGE platform", os.Args[0])
	}

	platform, err := v1.ParsePlatform(os.Args[3])
	if err != nil {
		log.Fatalf("%v", err)
	}

	image, err := load_single_arch(os.Args[2])
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, err = append_multi_arch(os.Args[1], image, platform)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
