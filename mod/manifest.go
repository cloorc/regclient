package mod

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
)

// WithAnnotation adds an annotation, or deletes it if the value is set to an empty string
func WithAnnotation(name, value string) Opts {
	return func(dc *dagConfig, dm *dagManifest) error {
		// TODO: use Get/Set Annotations methods on manifests
		dc.stepsManifest = append(dc.stepsManifest, func(ctx context.Context, rc *regclient.RegClient, r ref.Ref, dm *dagManifest) error {
			var err error
			changed := false
			// only annotate top manifest
			if dm.mod == deleted || !dm.top {
				return nil
			}
			om := dm.m.GetOrig()
			if dm.m.IsList() {
				ociI, err := manifest.OCIIndexFromAny(om)
				if err != nil {
					return err
				}
				if ociI.Annotations == nil {
					ociI.Annotations = map[string]string{}
				}
				cur, ok := ociI.Annotations[name]
				if value == "" && ok {
					delete(ociI.Annotations, name)
					changed = true
				} else if value != "" && value != cur {
					ociI.Annotations[name] = value
					changed = true
				}
				err = manifest.OCIIndexToAny(ociI, &om)
				if err != nil {
					return err
				}
			} else {
				ociM, err := manifest.OCIManifestFromAny(om)
				if err != nil {
					return err
				}
				if ociM.Annotations == nil {
					ociM.Annotations = map[string]string{}
				}
				cur, ok := ociM.Annotations[name]
				if value == "" && ok {
					delete(ociM.Annotations, name)
					changed = true
				} else if value != "" && value != cur {
					ociM.Annotations[name] = value
					changed = true
				}
				err = manifest.OCIManifestToAny(ociM, &om)
				if err != nil {
					return err
				}
			}
			if changed {
				dm.mod = replaced
				err = dm.m.SetOrig(om)
				if err != nil {
					return err
				}
				dm.newDesc = dm.m.GetDescriptor()
			}
			return nil
		})
		return nil
	}
}

// WithAnnotationOCIBase adds annotations for the base image
func WithAnnotationOCIBase(rBase ref.Ref, dBase digest.Digest) Opts {
	return func(dc *dagConfig, dm *dagManifest) error {
		dc.stepsManifest = append(dc.stepsManifest, func(ctx context.Context, rc *regclient.RegClient, r ref.Ref, dm *dagManifest) error {
			if dm.mod == deleted || !dm.top {
				return nil
			}
			annoBaseDig := "org.opencontainers.image.base.digest"
			annoBaseName := "org.opencontainers.image.base.name"
			changed := false
			om := dm.m.GetOrig()
			if dm.m.IsList() {
				ociI, err := manifest.OCIIndexFromAny(om)
				if err != nil {
					return err
				}
				if ociI.Annotations == nil {
					ociI.Annotations = map[string]string{}
				}
				if ociI.Annotations[annoBaseName] != rBase.CommonName() {
					ociI.Annotations[annoBaseName] = rBase.CommonName()
					changed = true
				}
				if ociI.Annotations[annoBaseDig] != dBase.String() {
					ociI.Annotations[annoBaseDig] = dBase.String()
					changed = true
				}
				err = manifest.OCIIndexToAny(ociI, &om)
				if err != nil {
					return err
				}
			} else {
				ociM, err := manifest.OCIManifestFromAny(om)
				if err != nil {
					return err
				}
				if ociM.Annotations == nil {
					ociM.Annotations = map[string]string{}
				}
				if ociM.Annotations[annoBaseName] != rBase.CommonName() {
					ociM.Annotations[annoBaseName] = rBase.CommonName()
					changed = true
				}
				if ociM.Annotations[annoBaseDig] != dBase.String() {
					ociM.Annotations[annoBaseDig] = dBase.String()
					changed = true
				}
				err = manifest.OCIManifestToAny(ociM, &om)
				if err != nil {
					return err
				}
			}
			if changed {
				dm.mod = replaced
				err := dm.m.SetOrig(om)
				if err != nil {
					return err
				}
				dm.newDesc = dm.m.GetDescriptor()
			}
			return nil
		})
		return nil
	}
}

// WithLabelToAnnotation copies image config labels to manifest annotations
func WithLabelToAnnotation() Opts {
	return func(dc *dagConfig, dm *dagManifest) error {
		dc.stepsManifest = append(dc.stepsManifest, func(c context.Context, rc *regclient.RegClient, r ref.Ref, dm *dagManifest) error {
			if dm.mod == deleted {
				return nil
			}
			changed := false
			if dm.m.IsList() {
				return nil
			}
			om := dm.m.GetOrig()
			ociOM, err := manifest.OCIManifestFromAny(om)
			if err != nil {
				return err
			}
			if ociOM.Annotations == nil {
				ociOM.Annotations = map[string]string{}
			}
			if dm.config == nil || dm.config.oc == nil {
				return nil
			}
			oc := dm.config.oc.GetConfig()
			if oc.Config.Labels == nil {
				return nil
			}
			for name, value := range oc.Config.Labels {
				cur, ok := ociOM.Annotations[name]
				if !ok || cur != value {
					ociOM.Annotations[name] = value
					changed = true
				}
			}
			if !changed {
				return nil
			}
			err = manifest.OCIManifestToAny(ociOM, &om)
			if err != nil {
				return err
			}
			err = dm.m.SetOrig(om)
			if err != nil {
				return err
			}
			dm.newDesc = dm.m.GetDescriptor()
			dm.mod = replaced
			return nil
		})
		return nil
	}
}

// WithExternalURLsRm strips external URLs from descriptors and adjusts media type to match
func WithExternalURLsRm() Opts {
	return func(dc *dagConfig, dm *dagManifest) error {
		dc.stepsManifest = append(dc.stepsManifest, func(ctx context.Context, rc *regclient.RegClient, r ref.Ref, dm *dagManifest) error {
			if dm.mod == deleted {
				return nil
			}
			changed := false
			if dm.m.IsList() {
				return nil
			}
			om := dm.m.GetOrig()
			ociOM, err := manifest.OCIManifestFromAny(om)
			if err != nil {
				return err
			}
			// strip layers from image
			for i := range ociOM.Layers {
				if len(ociOM.Layers[i].URLs) > 0 {
					ociOM.Layers[i].URLs = []string{}
					mt := ociOM.Layers[i].MediaType
					switch mt {
					case types.MediaTypeDocker2ForeignLayer:
						mt = types.MediaTypeDocker2LayerGzip
					case types.MediaTypeOCI1ForeignLayer:
						mt = types.MediaTypeOCI1Layer
					case types.MediaTypeOCI1ForeignLayerGzip:
						mt = types.MediaTypeOCI1LayerGzip
					case types.MediaTypeOCI1ForeignLayerZstd:
						mt = types.MediaTypeOCI1LayerZstd
					}
					ociOM.Layers[i].MediaType = mt
					changed = true
				}
			}
			// also strip from dag so other steps don't skip the external layer
			for i, dl := range dm.layers {
				if dl.mod == deleted {
					continue
				}
				if dl.newDesc.Digest == "" && len(dl.desc.URLs) > 0 {
					dl.newDesc = dl.desc
				}
				if len(dl.newDesc.URLs) > 0 {
					dl.newDesc.URLs = []string{}
					dm.layers[i] = dl
				}
			}
			if !changed {
				return nil
			}
			err = manifest.OCIManifestToAny(ociOM, &om)
			if err != nil {
				return err
			}
			err = dm.m.SetOrig(om)
			if err != nil {
				return err
			}
			dm.newDesc = dm.m.GetDescriptor()
			dm.mod = replaced
			return nil
		})
		return nil
	}
}

func WithRebase() Opts {
	return func(dc *dagConfig, dm *dagManifest) error {
		ma, ok := dm.m.(manifest.Annotator)
		if !ok {
			return fmt.Errorf("rebased failed, manifest does not support annotations")
		}
		annot, err := ma.GetAnnotations()
		if err != nil {
			return fmt.Errorf("failed getting annotations: %w", err)
		}
		baseName, ok := annot[types.AnnotationBaseImageName]
		if !ok {
			return fmt.Errorf("annotation for base image is missing (%s or %s)%.0w", types.AnnotationBaseImageName, types.AnnotationBaseImageDigest, types.ErrMissingAnnotation)
		}
		baseDigest, ok := annot[types.AnnotationBaseImageDigest]
		if !ok {
			return fmt.Errorf("annotation for base image is missing (%s or %s)%.0w", types.AnnotationBaseImageName, types.AnnotationBaseImageDigest, types.ErrMissingAnnotation)
		}
		rNew, err := ref.New(baseName)
		if err != nil {
			return fmt.Errorf("failed to parse base name: %w", err)
		}
		dig, err := digest.Parse(baseDigest)
		if err != nil {
			return fmt.Errorf("failed to parse base digest: %w", err)
		}
		rOld := rNew
		rOld.Digest = dig.String()

		return rebaseAddStep(dc, rOld, rNew)
	}
}

// WithRebaseRefs swaps the base image layers from the old to the new reference
func WithRebaseRefs(rOld, rNew ref.Ref) Opts {
	// cache old and new manifests, variable is nil until first pulled
	return func(dc *dagConfig, dm *dagManifest) error {
		return rebaseAddStep(dc, rOld, rNew)
	}
}

func rebaseAddStep(dc *dagConfig, rOld, rNew ref.Ref) error {
	var mbOldCache, mbNewCache manifest.Manifest
	dc.stepsManifest = append(dc.stepsManifest, func(ctx context.Context, rc *regclient.RegClient, r ref.Ref, dm *dagManifest) error {
		// skip if manifest list or deleted
		if dm.m.IsList() || dm.mod == deleted {
			return nil
		}
		// get and cache base manifests
		var err error
		if mbOldCache == nil {
			mbOldCache, err = rc.ManifestGet(ctx, rOld)
			if err != nil {
				return err
			}
		}
		if mbNewCache == nil {
			mbNewCache, err = rc.ManifestGet(ctx, rNew)
			if err != nil {
				return err
			}
		}
		// if old and new are the same, skip rebase
		if mbOldCache.GetDescriptor().Equal(mbNewCache.GetDescriptor()) {
			return nil
		}
		// get the platform from the config, pull the relevant manifest
		oc := dm.config.oc.GetConfig()
		p := platform.Platform{
			OS:           oc.OS,
			Architecture: oc.Architecture,
			Variant:      oc.Variant,
			OSVersion:    oc.OSVersion,
			OSFeatures:   oc.OSFeatures,
			Features:     oc.OSFeatures,
		}
		mbOld := mbOldCache
		if mbOld.IsList() {
			d, err := manifest.GetPlatformDesc(mbOld, &p)
			if err != nil {
				return err
			}
			rp := rOld
			rp.Digest = d.Digest.String()
			mbOld, err = rc.ManifestGet(ctx, rp)
			if err != nil {
				return err
			}
		}
		mbNew := mbNewCache
		if mbNew.IsList() {
			d, err := manifest.GetPlatformDesc(mbNew, &p)
			if err != nil {
				return err
			}
			rp := rNew
			rp.Digest = d.Digest.String()
			mbNew, err = rc.ManifestGet(ctx, rp)
			if err != nil {
				return err
			}
		}
		// load layers and config from image and old/new base images
		imageOld, ok := mbOld.(manifest.Imager)
		if !ok {
			return fmt.Errorf("base original image is not an image")
		}
		layersOld, err := imageOld.GetLayers()
		if err != nil {
			return err
		}
		cdOld, err := imageOld.GetConfig()
		if err != nil {
			return err
		}
		confOld, err := rc.BlobGetOCIConfig(ctx, rOld, cdOld)
		if err != nil {
			return err
		}
		confOCIOld := confOld.GetConfig()

		imageNew, ok := mbNew.(manifest.Imager)
		if !ok {
			return fmt.Errorf("base new image is not an image")
		}
		layersNew, err := imageNew.GetLayers()
		if err != nil {
			return err
		}
		cdNew, err := imageNew.GetConfig()
		if err != nil {
			return err
		}
		confNew, err := rc.BlobGetOCIConfig(ctx, rNew, cdNew)
		if err != nil {
			return err
		}
		confOCINew := confNew.GetConfig()

		mi, ok := dm.m.(manifest.Imager)
		if !ok {
			return fmt.Errorf("manifest is not an image")
		}
		layers, err := mi.GetLayers()
		if err != nil {
			return err
		}
		conf := dm.config.oc
		confOCI := conf.GetConfig()

		// validate current base
		if len(layersOld) > len(layers) {
			return fmt.Errorf("base image has more layers than modified image%.0w", types.ErrMismatch)
		}
		for i := range layersOld {
			if !layers[i].Same(layersOld[i]) {
				return fmt.Errorf("old base image does not match image layers, layer %d, base %v, image %v%.0w", i, layersOld[i], layers[i], types.ErrMismatch)
			}
		}
		if len(confOCIOld.History) > len(confOCI.History) {
			return fmt.Errorf("base image has more history entries than modified image%.0w", types.ErrMismatch)
		}
		historyLayers := 0
		for i := range confOCIOld.History {
			if confOCI.History[i].Author != confOCIOld.History[i].Author ||
				confOCI.History[i].Comment != confOCIOld.History[i].Comment ||
				!confOCI.History[i].Created.Equal(*confOCIOld.History[i].Created) ||
				confOCI.History[i].CreatedBy != confOCIOld.History[i].CreatedBy ||
				confOCI.History[i].EmptyLayer != confOCIOld.History[i].EmptyLayer {
				return fmt.Errorf("old base image does not match image history, entry %d, base %v, image %v%.0w", i, confOCIOld.History[i], confOCI.History[i], types.ErrMismatch)
			}
			if !confOCIOld.History[i].EmptyLayer {
				historyLayers++
			}
		}
		if len(layersOld) != historyLayers || len(layersOld) != len(confOCIOld.RootFS.DiffIDs) {
			return fmt.Errorf("old base image layer count doesn't match history%.0w", types.ErrMismatch)
		}
		if len(confOCIOld.RootFS.DiffIDs) > len(confOCI.RootFS.DiffIDs) {
			return fmt.Errorf("base image has more rootfs entries than modified image%.0w", types.ErrMismatch)
		}
		for i := range confOCIOld.RootFS.DiffIDs {
			if confOCI.RootFS.DiffIDs[i] != confOCIOld.RootFS.DiffIDs[i] {
				return fmt.Errorf("old base image does not match image rootfs, entry %d, base %s, image %s%.0w", i, confOCIOld.RootFS.DiffIDs[i].String(), confOCI.RootFS.DiffIDs[i].String(), types.ErrMismatch)
			}
		}
		// validate new base
		historyLayers = 0
		for i := range confOCINew.History {
			if !confOCINew.History[i].EmptyLayer {
				historyLayers++
			}
		}
		if len(layersNew) != historyLayers || len(confOCINew.RootFS.DiffIDs) != historyLayers {
			return fmt.Errorf("new base image config history doesn't match layer count")
		}
		// copy blobs from new base to repo
		for _, d := range layersNew {
			if err := rc.BlobCopy(ctx, rNew, r, d); err != nil {
				return fmt.Errorf("failed copying blobs for rebase: %w", err)
			}
		}

		// delete the old layers and config entries from vars and dag
		pruneNum := len(layersOld)
		i := 0
		for pruneNum > 0 {
			if dm.layers[i].mod == added {
				i++
				continue
			}
			if i == 0 {
				dm.layers = dm.layers[1:]
			} else if i >= len(dm.layers)-1 {
				dm.layers = dm.layers[:i]
			} else {
				dm.layers = append(dm.layers[:i], dm.layers[i+1:]...)
			}
			pruneNum--
		}
		layers = layers[len(layersOld):]
		confOCI.History = confOCI.History[len(confOCIOld.History):]
		confOCI.RootFS.DiffIDs = confOCI.RootFS.DiffIDs[len(confOCIOld.RootFS.DiffIDs):]

		// insert new layers and config entries in vars and dag, mark as modified
		dagAdd := []*dagLayer{}
		for i, l := range layersNew {
			dagAdd = append(dagAdd, &dagLayer{
				mod:      unchanged,
				ucDigest: confOCINew.RootFS.DiffIDs[i],
				desc:     l,
			})
		}
		dm.layers = append(dagAdd, dm.layers...)
		layers = append(layersNew, layers...)
		err = mi.SetLayers(layers)
		if err != nil {
			return err
		}
		confOCI.History = append(confOCINew.History, confOCI.History...)
		confOCI.RootFS.DiffIDs = append(confOCINew.RootFS.DiffIDs, confOCI.RootFS.DiffIDs...)
		dm.config.oc.SetConfig(confOCI)
		dm.config.newDesc = dm.config.oc.GetDescriptor()

		// set modified flags on config and manifest
		dm.config.modified = true
		dm.mod = replaced

		return nil
	})
	return nil
}
