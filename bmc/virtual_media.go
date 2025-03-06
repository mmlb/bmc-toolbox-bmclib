package bmc

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type VirtualMediaConfig struct {
	Image          string
	MediaType      string
	Password       string
	UserName       string
	Inserted       bool
	WriteProtected bool
}

// VirtualMediaGetter gets the virtual media attached to a machine
type VirtualMediaGetter interface {
	GetVirtualMedia(ctx context.Context) (medias []VirtualMediaConfig, err error)
}

// VirtualMediaSetter controls the virtual media attached to a machine
type VirtualMediaSetter interface {
	SetVirtualMedia(ctx context.Context, kind, mediaURL string) (ok bool, err error)
}

// VirtualMediaProviders is an internal struct to correlate an implementation/provider and its name
type virtualMediaProviders struct {
	name               string
	virtualMediaSetter VirtualMediaSetter
}

// GetVirtualMediaFromInterfaces identifies implementations of the VirtualMediaGetter interface and queries them for any VirtualMedia that have been configured
func GetVirtualMediaFromInterfaces(ctx context.Context, generic []interface{}) (Metadata, []VirtualMediaConfig, error) {
	var md Metadata
	var err error
	for _, elem := range generic {
		if elem == nil {
			continue
		}

		name := getProviderName(elem)
		getter, ok := elem.(VirtualMediaGetter)
		if !ok {
			e := fmt.Sprintf("not a VirtualMediaGetter implementation: %T", elem)
			err = multierror.Append(err, errors.New(e))
			continue
		}

		select {
		case <-ctx.Done():
			return md, nil, multierror.Append(err, ctx.Err())
		default:
			md.ProvidersAttempted = append(md.ProvidersAttempted, name)

			medias, err := getter.GetVirtualMedia(ctx)
			if err != nil {
				err = multierror.Append(err, errors.WithMessagef(err, "provider: %v", name))
				continue
			}

			if len(medias) == 0 {
				err = multierror.Append(err, fmt.Errorf("provider: %v, failed to get virtual media", name))
				continue
			}

			md.SuccessfulProvider = name
			return md, medias, nil
		}
	}

	return md, nil, multierror.Append(err, errors.New("failed to get virtual media"))
}

// setVirtualMedia sets the virtual media.
func setVirtualMedia(ctx context.Context, kind, mediaURL string, b []virtualMediaProviders) (ok bool, metadata Metadata, err error) {
	var metadataLocal Metadata

	for _, elem := range b {
		if elem.virtualMediaSetter == nil {
			continue
		}
		select {
		case <-ctx.Done():
			err = multierror.Append(err, ctx.Err())

			return false, metadata, err
		default:
			metadataLocal.ProvidersAttempted = append(metadataLocal.ProvidersAttempted, elem.name)
			ok, setErr := elem.virtualMediaSetter.SetVirtualMedia(ctx, kind, mediaURL)
			if setErr != nil {
				err = multierror.Append(err, errors.WithMessagef(setErr, "provider: %v", elem.name))
				continue
			}
			if !ok {
				err = multierror.Append(err, fmt.Errorf("provider: %v, failed to set virtual media", elem.name))
				continue
			}
			metadataLocal.SuccessfulProvider = elem.name
			return ok, metadataLocal, nil
		}
	}
	return ok, metadataLocal, multierror.Append(err, errors.New("failed to set virtual media"))
}

// SetVirtualMediaFromInterfaces identifies implementations of the virtualMediaSetter interface and passes the found implementations to the setVirtualMedia() wrapper
func SetVirtualMediaFromInterfaces(ctx context.Context, kind, mediaURL string, generic []interface{}) (ok bool, metadata Metadata, err error) {
	bdSetters := make([]virtualMediaProviders, 0)
	for _, elem := range generic {
		if elem == nil {
			continue
		}
		temp := virtualMediaProviders{name: getProviderName(elem)}
		switch p := elem.(type) {
		case VirtualMediaSetter:
			temp.virtualMediaSetter = p
			bdSetters = append(bdSetters, temp)
		default:
			e := fmt.Sprintf("not a VirtualMediaSetter implementation: %T", p)
			err = multierror.Append(err, errors.New(e))
		}
	}
	if len(bdSetters) == 0 {
		return ok, metadata, multierror.Append(err, errors.New("no VirtualMediaSetter implementations found"))
	}
	return setVirtualMedia(ctx, kind, mediaURL, bdSetters)
}
