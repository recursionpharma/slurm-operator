// Command publish-image rebuilds one of the slurm-operator container images
// (manager or webhook) from a prebuilt linux/amd64 binary, so the images we
// push to our registry can be reproduced by anyone with go + network access.
//
// Usage: publish-image <manager|webhook> <binary> <base-ref> <dest-ref>
//
// It wraps the binary in a layer on top of base-ref (the distroless base,
// pinned by digest in build.sh) and sets the same container config as the
// upstream Dockerfile: USER 65532:65532, WORKDIR /, ENTRYPOINT ["/<binary>"].
// Auth uses standard docker credentials (see build.sh).
package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: publish-image <manager|webhook> <binary> <base-ref> <dest-ref>")
		os.Exit(2)
	}
	bin, binPath, baseRef, destRef := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	if bin != "manager" && bin != "webhook" {
		fmt.Fprintln(os.Stderr, "binary must be 'manager' or 'webhook'")
		os.Exit(2)
	}

	base, err := name.ParseReference(baseRef)
	must(err)
	dest, err := name.ParseReference(destRef)
	must(err)

	img, err := remote.Image(base, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	must(err)

	var buf bytes.Buffer
	if err := writeSingleFileLayer(&buf, bin, binPath); err != nil {
		must(err)
	}
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
	must(err)

	img, err = mutate.Append(img, mutate.Addendum{Layer: layer})
	must(err)

	cfgFile, err := img.ConfigFile()
	must(err)
	cfg := cfgFile.Config
	cfg.User = "65532:65532"
	cfg.WorkingDir = "/"
	cfg.Entrypoint = []string{"/" + bin}
	cfg.Cmd = nil
	img, err = mutate.ConfigFile(img, &v1.ConfigFile{Config: cfg, Architecture: "amd64", OS: "linux", RootFS: cfgFile.RootFS})
	must(err)

	must(remote.Write(dest, img, remote.WithAuthFromKeychain(authn.DefaultKeychain)))
	digest, err := img.Digest()
	must(err)
	fmt.Printf("%s: pushed %s (digest %s)\n", bin, dest.Name(), digest)
}

// writeSingleFileLayer writes a minimal OCI layer tar containing one root-owned
// 0755 regular file at the archive root named name, with the contents of path.
func writeSingleFileLayer(w *bytes.Buffer, name, path string) error {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	fi, err := f.Stat()
	must(err)

	tw := tar.NewWriter(w)
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     fi.Size(),
		Mode:     0o755,
		Uid:      0,
		Gid:      0,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}
	return tw.Close()
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
