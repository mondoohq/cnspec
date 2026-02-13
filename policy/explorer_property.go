// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/checksums"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/mrn"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/multierr"
)

type PropertyRef struct {
	*Property
	Name string
	Mrn  string
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
func (p *Property) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, MRN_RESOURCE_PROPERTY, p.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return multierr.Wrap(err, "failed to refresh mrn for query "+p.Title)
	}
	p.Mrn = nu
	p.Uid = ""

	return nil
}

// Compile a given property and return the bundle.
func (p *Property) Compile(props mqlc.PropsHandler, conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	return mqlc.Compile(p.Mql, props, conf)
}

// id gets any valid ID for the property, prioritizing uid > mrn > title
func (p *Property) id() string {
	if p.Uid != "" {
		return p.Uid
	}
	if p.Mrn != "" {
		return p.Mrn
	}
	return p.Title
}

// RefreshChecksumAndType by compiling the query and updating the Checksum field
func (p *Property) RefreshChecksumAndType(conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	var bundle *llx.CodeBundle

	if p.Mql == "" && p.Type == "" {
		return nil, errors.New("property must not be empty or must define a type (property '" + p.id() + "')")
	}

	if p.Mql != "" {
		var err error
		bundle, err = p.Compile(nil, conf)
		if err != nil {
			return bundle, multierr.Wrap(err, "failed to compile property '"+p.id()+"', mql: '"+p.Mql+"'")
		}

		if bundle.GetCodeV2().GetId() == "" {
			return bundle, errors.New("failed to compile query: received empty result values")
		}

		p.CodeId = bundle.CodeV2.Id
		p.Mql = bundle.Source

		if entrypoints := bundle.CodeV2.Entrypoints(); len(entrypoints) == 1 {
			ep := entrypoints[0]
			chunk := bundle.CodeV2.Chunk(ep)
			typ := chunk.Type()
			p.Type = string(typ)
		} else {
			p.Type = string(types.Any)
		}
	}

	c := checksums.New.
		Add(p.Mql).
		Add(p.CodeId).
		Add(p.Mrn).
		Add(p.Type).
		Add(p.Context).
		Add(p.Title).Add("v2").
		Add(p.Desc)

	for i := range p.For {
		f := p.For[i]
		c = c.Add(f.Mrn)
	}

	p.Checksum = c.String()

	return bundle, nil
}

func (p *Property) Merge(base *Property) {
	if p.Mql == "" {
		p.Mql = base.Mql
		p.CodeId = base.CodeId
	}
	if p.Type == "" {
		p.Type = base.Type
	}
	if p.Context == "" {
		p.Context = base.Context
	}
	if p.Title == "" {
		p.Title = base.Title
	}
	if p.Desc == "" {
		p.Desc = base.Desc
	}
	if len(p.For) == 0 {
		p.For = base.For
	}
}

func (p *Property) Clone() *Property {
	res := &Property{
		Mql:      p.Mql,
		CodeId:   p.CodeId,
		Checksum: p.Checksum,
		Mrn:      p.Mrn,
		Uid:      p.Uid,
		Type:     p.Type,
		Context:  p.Context,
		Title:    p.Title,
		Desc:     p.Desc,
	}

	if p.For != nil {
		res.For = make([]*ObjectRef, len(p.For))
		for i := range p.For {
			fr := p.For[i]
			res.For[i] = &ObjectRef{
				Mrn: fr.Mrn,
				Uid: fr.Uid,
			}
		}
	}

	return res
}

type PropsCache struct {
	cache        map[string]*Property
	uidOnlyProps map[string]*Property
}

func NewPropsCache() PropsCache {
	return PropsCache{
		cache:        map[string]*Property{},
		uidOnlyProps: map[string]*Property{},
	}
}

// Add properties, NOT overwriting existing ones (instead we add them as base)
func (c PropsCache) Add(props ...*Property) {
	for i := range props {
		base := props[i]

		if base.Uid != "" && base.Mrn == "" {
			c.uidOnlyProps[base.Uid] = base
			continue
		}

		merged := base

		if base.Mrn != "" {
			var name string
			name, _ = GetPropName(base.Mrn)
			if uidProp, ok := c.uidOnlyProps[name]; ok {
				p := uidProp.Clone()
				p.Merge(base)
				base = p
				merged = p
			}

			if existingProp, ok := c.cache[base.Mrn]; ok {
				existingProp.Merge(base)
				merged = existingProp
			} else {
				c.cache[base.Mrn] = base
			}
		}

		for i := range base.For {
			pfor := base.For[i]
			if pfor.Mrn != "" {
				if existingProp, ok := c.cache[pfor.Mrn]; ok {
					existingProp.Merge(merged)
				} else {
					c.cache[pfor.Mrn] = merged.CloneVT()
				}
			}
		}
	}
}

// Get tries to get the mrn, will also return uid-based
// properties if they exist first
func (c PropsCache) Get(propMrn string) (*Property, string, error) {
	if res, ok := c.cache[propMrn]; ok {
		name, err := GetPropName(propMrn)
		if err != nil {
			return nil, "", errors.New("failed to get property name")
		}
		if uidProp, ok := c.uidOnlyProps[name]; ok {
			p := uidProp.Clone()
			p.Merge(res)
			return p, name, nil
		}
		return res, name, nil
	}

	return nil, "", errors.New("property " + propMrn + " not found")
}

func GetPropName(propMrn string) (string, error) {
	name, err := mrn.GetResource(propMrn, MRN_RESOURCE_PROPERTY)
	if err != nil {
		name, err = mrn.GetResource(propMrn, MRN_RESOURCE_QUERY)
		if err != nil {
			return "", err
		}
	}

	return name, nil
}
