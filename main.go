//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"

	_ "crypto/sha256"
)

const tagStaged = "staged"

func appendMultiArch(path string, image v1.Image, platform *v1.Platform) (v1.ImageIndex, error) {

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

func loadSingleArch(path string) (v1.Image, error) {
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

func loadFiles(ctx context.Context, store *file.Store, tarball string) ([]ocispec.Descriptor, error) {
	filename, mediaType, _ := parseFileRef(tarball, "")
	name := filepath.Clean(filename)
	if !filepath.IsAbs(name) {
		// convert to slash-separated path unless it is absolute path
		name = filepath.ToSlash(name)
	}
	file, err := store.Add(ctx, name, mediaType, filename)
	if err != nil {
		return nil, err
	}
	return []ocispec.Descriptor{file}, nil
}

func packManifest(ctx context.Context, store *file.Store, filename string) (ocispec.Descriptor, error) {
	files, err := loadFiles(ctx, store, filename)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	var packOpts oras.PackOptions
	manifestDesc, err := oras.Pack(ctx, store, files, packOpts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	if err := store.Tag(ctx, manifestDesc, tagStaged); err != nil {
		return ocispec.Descriptor{}, err
	}
	return manifestDesc, nil
}

func parseFileRef(ref string, mediaType string) (string, string, string) {
	i := strings.LastIndex(ref, "@")

	platform := ""
	if i >= 0 {
		platform = ref[i+1:]
		ref = ref[:i]
	}

	i = strings.LastIndex(ref, ":")
	if i < 0 {
		return ref, mediaType, platform
	}
	return ref[:i], ref[i+1:], platform
}

func createOCIFromTarball(dir, filename string) error {
	ctx := context.Background()

	dst, err := oci.New(dir)
	if err != nil {
		return err
	}

	// Prepare manifest
	store := file.New("")
	defer store.Close()

	// Ready to push
	_, err = packManifest(ctx, store, filename)
	if err != nil {
		return err
	}

	_, err = oras.Copy(ctx, store, tagStaged, dst, tagStaged)
	return err
}

func mappendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  fmt.Sprintf("%s image tarball platform", os.Args[0]),
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := v1.ParsePlatform(args[2])
			if err != nil {
				return err
			}

			dir, err := ioutil.TempDir("", "oci")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)

			err = createOCIFromTarball(dir, args[1])
			if err != nil {
				return err
			}

			image, err := loadSingleArch(dir)
			if err != nil {
				return err
			}

			_, err = appendMultiArch(args[0], image, platform)
			return err
		},
	}
	return cmd

}

func main() {
	cmd := mappendCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
