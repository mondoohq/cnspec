// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"io"
)

type MultiDecoder struct {
	decoders []Decoder
}

func DefaultMultiDecoder() MultiDecoder {
	return NewMultiDecoder(
		NewCycloneDX(FormatCycloneDxJSON),
		NewCycloneDX(FormatCycloneDxXML),
		NewSPDX(FormatSpdxTagValue),
		NewSPDX(FormatSpdxJSON),
		New(FormatJson),
	)
}

func NewMultiDecoder(decoders ...Decoder) MultiDecoder {
	return MultiDecoder{decoders: decoders}
}

func (m MultiDecoder) Parse(r io.ReadSeeker) (*Sbom, error) {
	if len(m.decoders) == 0 {
		return nil, errors.New("no decoders available in multi decoder")
	}

	var decoderErr error
	for _, decoder := range m.decoders {
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		s, err := decoder.Parse(r)
		if err == nil {
			return s, nil
		}
		decoderErr = err
	}
	return nil, decoderErr
}
