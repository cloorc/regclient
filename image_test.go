package regclient

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/regclient/regclient/internal/rwfs"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/ref"
)

func TestImageCheckBase(t *testing.T) {
	ctx := context.Background()
	fsOS := rwfs.OSNew("")
	fsMem := rwfs.MemNew()
	err := rwfs.CopyRecursive(fsOS, "testdata", fsMem, ".")
	if err != nil {
		t.Errorf("failed to setup memfs copy: %v", err)
		return
	}
	delayInit, _ := time.ParseDuration("0.05s")
	delayMax, _ := time.ParseDuration("0.10s")
	rc := New(WithFS(fsMem), WithRetryDelay(delayInit, delayMax))
	rb1, err := ref.New("ocidir://testrepo:b1")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}
	rb2, err := ref.New("ocidir://testrepo:b2")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}
	rb3, err := ref.New("ocidir://testrepo:b3")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}
	m3, err := rc.ManifestHead(ctx, rb3)
	if err != nil {
		t.Errorf("failed to get digest for base3: %v", err)
		return
	}
	dig3 := m3.GetDescriptor().Digest
	r1, err := ref.New("ocidir://testrepo:v1")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}
	r2, err := ref.New("ocidir://testrepo:v2")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}
	r3, err := ref.New("ocidir://testrepo:v3")
	if err != nil {
		t.Errorf("failed to setup ref: %v", err)
		return
	}

	tests := []struct {
		name      string
		opts      []ImageOpts
		r         ref.Ref
		expectErr error
	}{
		{
			name:      "missing annotation",
			r:         r1,
			expectErr: types.ErrMissingAnnotation,
		},
		{
			name:      "annotation v2",
			r:         r2,
			expectErr: types.ErrMismatch,
		},
		{
			name:      "annotation v3",
			r:         r3,
			expectErr: types.ErrMismatch,
		},
		{
			name: "manual v2, b1",
			r:    r2,
			opts: []ImageOpts{ImageWithCheckBaseRef(rb1.CommonName())},
		},
		{
			name:      "manual v2, b2",
			r:         r2,
			opts:      []ImageOpts{ImageWithCheckBaseRef(rb2.CommonName())},
			expectErr: types.ErrMismatch,
		},
		{
			name:      "manual v2, b3",
			r:         r2,
			opts:      []ImageOpts{ImageWithCheckBaseRef(rb3.CommonName())},
			expectErr: types.ErrMismatch,
		},
		{
			name: "manual v3, b1",
			r:    r3,
			opts: []ImageOpts{ImageWithCheckBaseRef(rb1.CommonName())},
		},
		{
			name: "manual v3, b3 with digest",
			r:    r3,
			opts: []ImageOpts{ImageWithCheckBaseRef(rb3.CommonName()), ImageWithCheckBaseDigest(dig3.String())},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rc.ImageCheckBase(ctx, tt.r, tt.opts...)
			if tt.expectErr != nil {
				if err == nil {
					t.Errorf("check base did not fail")
				} else if err.Error() != tt.expectErr.Error() && !errors.Is(err, tt.expectErr) {
					t.Errorf("error mismatch, expected %v, received %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("check base failed")
				}
			}
		})
	}
}

func TestCopy(t *testing.T) {
	ctx := context.Background()
	// create regclient
	delayInit, _ := time.ParseDuration("0.05s")
	delayMax, _ := time.ParseDuration("0.10s")
	rc := New(WithRetryDelay(delayInit, delayMax))
	tempDir := t.TempDir()
	rSrc, err := ref.New("ocidir://./testdata/testrepo:v1")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err := ref.New("ocidir://" + tempDir + ":v1")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt)
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
	rSrc, err = ref.New("ocidir://./testdata/testrepo:v2")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err = ref.New("ocidir://" + tempDir + ":v2")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt, ImageWithReferrers(), ImageWithDigestTags())
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
	rSrc, err = ref.New("ocidir://./testdata/testrepo:v3")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err = ref.New("ocidir://" + tempDir + ":v3")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt)
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
	rSrc, err = ref.New("ocidir://./testdata/testrepo:v3")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err = ref.New("ocidir://" + tempDir + ":v3")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt, ImageWithReferrers(), ImageWithDigestTags(), ImageWithFastCheck())
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
	rSrc, err = ref.New("ocidir://./testdata/testrepo:child")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err = ref.New("ocidir://" + tempDir + ":child")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt, ImageWithReferrers(), ImageWithDigestTags())
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
	rSrc, err = ref.New("ocidir://./testdata/testrepo:mirror")
	if err != nil {
		t.Errorf("failed to parse src ref: %v", err)
	}
	rTgt, err = ref.New("ocidir://" + tempDir + ":mirror")
	if err != nil {
		t.Errorf("failed to parse tgt ref: %v", err)
	}
	err = rc.ImageCopy(ctx, rSrc, rTgt, ImageWithDigestTags())
	if err != nil {
		t.Errorf("failed to copy: %v", err)
	}
}

func TestExportImport(t *testing.T) {
	ctx := context.Background()
	// copy testdata images into memory
	fsOS := rwfs.OSNew("")
	fsMem := rwfs.MemNew()
	err := rwfs.CopyRecursive(fsOS, "testdata", fsMem, ".")
	if err != nil {
		t.Errorf("failed to setup memfs copy: %v", err)
		return
	}
	// create regclient
	delayInit, _ := time.ParseDuration("0.05s")
	delayMax, _ := time.ParseDuration("0.10s")
	rc := New(WithFS(fsMem), WithRetryDelay(delayInit, delayMax))
	rIn1, err := ref.New("ocidir://testrepo:v1")
	if err != nil {
		t.Errorf("failed to parse ref: %v", err)
	}
	rOut1, err := ref.New("ocidir://testout:v1")
	if err != nil {
		t.Errorf("failed to parse ref: %v", err)
	}
	rIn3, err := ref.New("ocidir://testrepo:v3")
	if err != nil {
		t.Errorf("failed to parse ref: %v", err)
	}
	rOut3, err := ref.New("ocidir://testout:v3")
	if err != nil {
		t.Errorf("failed to parse ref: %v", err)
	}

	// export repo to tar
	fileOut1, err := fsMem.Create("test1.tar")
	if err != nil {
		t.Errorf("failed to create output tar: %v", err)
	}
	err = rc.ImageExport(ctx, rIn1, fileOut1)
	fileOut1.Close()
	if err != nil {
		t.Errorf("failed to export: %v", err)
	}
	fileOut3, err := fsMem.Create("test3.tar.gz")
	if err != nil {
		t.Errorf("failed to create output tar: %v", err)
	}
	err = rc.ImageExport(ctx, rIn3, fileOut3, ImageWithExportCompress())
	fileOut3.Close()
	if err != nil {
		t.Errorf("failed to export: %v", err)
	}

	// modify tar for tests
	fileR, err := fsMem.Open("test1.tar")
	if err != nil {
		t.Errorf("failed to open tar: %v", err)
	}
	fileW, err := fsMem.Create("test2.tar")
	if err != nil {
		t.Errorf("failed to create tar: %v", err)
	}
	tr := tar.NewReader(fileR)
	tw := tar.NewWriter(fileW)
	for {
		th, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Errorf("failed to read tar header: %v", err)
		}
		th.Name = "./" + th.Name
		err = tw.WriteHeader(th)
		if err != nil {
			t.Errorf("failed to write tar header: %v", err)
		}
		if th.Size > 0 {
			_, err = io.Copy(tw, tr)
			if err != nil {
				t.Errorf("failed to copy tar file contents %s: %v", th.Name, err)
			}
		}
	}
	fileR.Close()
	fileW.Close()

	// import tar to repo
	fileIn2, err := fsMem.Open("test2.tar")
	if err != nil {
		t.Errorf("failed to open tar: %v", err)
	}
	fileIn2Seeker, ok := fileIn2.(io.ReadSeeker)
	if !ok {
		t.Fatalf("could not convert fileIn to io.ReadSeeker, type %T", fileIn2)
	}
	err = rc.ImageImport(ctx, rOut1, fileIn2Seeker)
	if err != nil {
		t.Errorf("failed to import: %v", err)
	}

	fileIn3, err := fsMem.Open("test3.tar.gz")
	if err != nil {
		t.Errorf("failed to open tar: %v", err)
	}
	fileIn3Seeker, ok := fileIn3.(io.ReadSeeker)
	if !ok {
		t.Fatalf("could not convert fileIn to io.ReadSeeker, type %T", fileIn3)
	}
	err = rc.ImageImport(ctx, rOut3, fileIn3Seeker)
	if err != nil {
		t.Errorf("failed to import: %v", err)
	}
}
