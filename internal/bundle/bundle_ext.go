// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"sort"
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/explorer"
	"go.mondoo.com/cnquery/v12/utils/timex"
	"gopkg.in/yaml.v3"
)

// SortContents the queries, policies and queries' variants in the bundle.
func (p *Bundle) SortContents() {
	sort.SliceStable(p.Queries, func(i, j int) bool {
		if p.Queries[i].Mrn == "" || p.Queries[j].Mrn == "" {
			return p.Queries[i].Uid < p.Queries[j].Uid
		}
		return p.Queries[i].Mrn < p.Queries[j].Mrn
	})

	sort.SliceStable(p.Policies, func(i, j int) bool {
		if p.Policies[i].Mrn == "" || p.Policies[j].Mrn == "" {
			return p.Policies[i].Uid < p.Policies[j].Uid
		}
		return p.Policies[i].Mrn < p.Policies[j].Mrn
	})

	for _, q := range p.Queries {
		sort.SliceStable(q.Variants, func(i, j int) bool {
			if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
				return q.Variants[i].Uid < q.Variants[j].Uid
			}
			return q.Variants[i].Mrn < q.Variants[j].Mrn
		})
	}
	for _, pl := range p.Policies {
		for _, g := range pl.Groups {
			for _, q := range g.Queries {
				sort.SliceStable(q.Variants, func(i, j int) bool {
					if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
						return q.Variants[i].Uid < q.Variants[j].Uid
					}
					return q.Variants[i].Mrn < q.Variants[j].Mrn
				})
			}
			for _, c := range g.Checks {
				sort.SliceStable(c.Variants, func(i, j int) bool {
					if c.Variants[i].Mrn == "" || c.Variants[j].Mrn == "" {
						return c.Variants[i].Uid < c.Variants[j].Uid
					}
					return c.Variants[i].Mrn < c.Variants[j].Mrn
				})
			}
		}
	}
}

func (x *Impact) UnmarshalYAML(node *yaml.Node) error {
	defer x.addFileContext(node)

	var res int32
	if err := node.Decode(&res); err == nil {
		x.Value = &ImpactValue{Value: res}
		return nil
	}

	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Impact
	if err := node.Decode((*tmp)(x)); err != nil {
		return err
	}

	return nil
}

func (v *Impact) MarshalYAML() (any, error) {
	if explorer.Action(v.Action) == explorer.Action_UNSPECIFIED && v.Scoring == explorer.ScoringSystem_SCORING_UNSPECIFIED && v.Weight < 1 {
		if v.Value == nil {
			return nil, nil
		}
		return v.Value.Value, nil
	}
	return v, nil
}

func (x *ImpactValue) UnmarshalYAML(node *yaml.Node) error {
	x.addFileContext(node)
	var res int32
	if err := node.Decode(&res); err == nil {
		x.Value = res
		return nil
	}

	type tmp ImpactValue
	if err := node.Decode((*tmp)(x)); err != nil {
		return errors.Wrap(err, "can't unmarshal impact value")
	}
	return nil
}

func (x *Filters) UnmarshalYAML(node *yaml.Node) error {
	x.addFileContext(node)

	var str string
	err := node.Decode(&str)
	if err == nil {
		x.Items = map[string]*Mquery{}
		x.Items[""] = &Mquery{
			Mql: str,
		}
		return nil
	}

	// FIXME: DEPRECATED, remove in v9.0 vv
	// This old style of specifying filters is going to be removed, we
	// have an alternative with list and keys
	var arr []string
	err = node.Decode(&arr)
	if err == nil {
		x.Items = map[string]*Mquery{}
		for i := range arr {
			x.Items[strconv.Itoa(i)] = &Mquery{Mql: arr[i]}
		}
		return nil
	}
	// ^^

	var list []*Mquery
	err = node.Decode(&list)
	if err == nil {
		x.Items = map[string]*Mquery{}
		for i := range list {
			x.Items[strconv.Itoa(i)] = list[i]
		}
		return nil
	}

	type tmp Filters
	if err := node.Decode((*tmp)(x)); err != nil {
		return errors.Wrap(err, "can't unmarshal filters")
	}
	return nil
}

func (v *Filters) MarshalYAML() (any, error) {
	if v.Items == nil {
		return nil, nil
	}

	res := make([]*Mquery, len(v.Items))
	i := 0
	for _, v := range v.Items {
		res[i] = v
		i++
	}

	if len(res) == 1 {
		return res[0].Mql, nil
	}

	return res, nil
}

func (x *Remediation) UnmarshalYAML(node *yaml.Node) error {
	x.addFileContext(node)

	var str string
	err := node.Decode(&str)
	if err == nil {
		x.Items = []*TypedDoc{{Id: "default", Desc: str}}
		return nil
	}

	// decode a slice of remediation types
	if err := node.Decode(&x.Items); err == nil {
		return nil
	}

	type tmp Remediation
	if err := node.Decode((*tmp)(x)); err != nil {
		return errors.Wrap(err, "can't unmarshal remediation")
	}
	return nil
}

func (x *Remediation) MarshalYAML() (any, error) {
	if len(x.Items) == 0 {
		return nil, nil
	}

	if len(x.Items) == 1 && x.Items[0].Id == "default" {
		return x.Items[0].Desc, nil
	}

	return x.Items, nil
}

func (x *RiskMagnitude) UnmarshalYAML(node *yaml.Node) error {
	x.addFileContext(node)

	var res float32
	if err := node.Decode(&res); err == nil {
		x.Value = res
		return nil
	}

	type tmp RiskMagnitude
	if err := node.Decode((*tmp)(x)); err != nil {
		return errors.Wrap(err, "can't unmarshal risk magnitude")
	}
	return nil
}

func (x *HumanTime) UnmarshalYAML(node *yaml.Node) error {
	x.addFileContext(node)

	var i int64
	if err := node.Decode(&i); err == nil {
		x.Seconds = i
		return nil
	}

	var s string
	if err := node.Decode(&s); err != nil {
		return errors.New("failed to parse " + string(node.Value) + " as a time string: " + err.Error())
	}

	v, err := timex.Parse(s, "")
	if err != nil {
		return errors.New("failed to parse " + s + " as time: " + err.Error())
	}

	x.Seconds = v.Unix()
	return nil
}
