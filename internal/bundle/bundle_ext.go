package bundle

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/explorer"
	"gopkg.in/yaml.v3"
)

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

func (v *Impact) MarshalYAML() (interface{}, error) {
	if v.Action == explorer.Action_UNSPECIFIED && v.Scoring == explorer.ScoringSystem_SCORING_UNSPECIFIED && v.Weight < 1 {
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

func (v *Filters) MarshalYAML() (interface{}, error) {
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

	err = node.Decode(x.Items)
	if err == nil {
		return nil
	}

	type tmp Remediation
	if err := node.Decode((*tmp)(x)); err != nil {
		return errors.Wrap(err, "can't unmarshal remediation")
	}
	return nil
}

func (x *Remediation) MarshalYAML() (interface{}, error) {
	if len(x.Items) == 0 {
		return nil, nil
	}

	if len(x.Items) == 1 && x.Items[0].Id == "default" {
		return x.Items[0].Desc, nil
	}

	return x.Items, nil
}
