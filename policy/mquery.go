package policy

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources/packs/os/info"
	"go.mondoo.com/cnquery/types"
)

// Compile a given query and return the bundle. Both v1 and v2 versions are compiled.
// Both versions will be given the same code id.
func (m *Mquery) Compile(props map[string]*llx.Primitive, mustCompileV1 bool) (*llx.CodeBundle, error) {
	if m.Query == "" {
		return nil, errors.New("query is not implemented '" + m.Mrn + "'")
	}

	schema := info.Registry.Schema()

	v2Code, err := mqlc.Compile(m.Query, schema,
		cnquery.Features{byte(cnquery.PiperCode)}, props)
	if err != nil {
		return nil, err
	}

	v1Code, err := mqlc.Compile(m.Query, schema,
		cnquery.Features{}, props)
	if err != nil {
		log.Debug().Err(err).Str("query", m.Query).Msg("query only compiles with piper code")
		if mustCompileV1 {
			return nil, err
		}
	} else {
		v2Code.DeprecatedV5Code = v1Code.GetDeprecatedV5Code()
	}

	if v2Code.DeprecatedV5Code != nil {
		v2Code.CodeV2.Id = v2Code.DeprecatedV5Code.Id
		if v2Code.GetLabels().GetLabels() == nil {
			v2Code.Labels = v1Code.Labels
		} else {
			for k, v := range v1Code.Labels.GetLabels() {
				v2Code.Labels.Labels[k] = v
			}
		}
		v2Code.DeprecatedV5Assertions = v1Code.GetDeprecatedV5Assertions()
	}
	return v2Code, nil
}

// RefreshAsAssetFilter filters treats this query as an asset filter and sets its Mrn, Title, and Checksum
func (m *Mquery) RefreshAsAssetFilter(mrn string) (*llx.CodeBundle, error) {
	bundle, err := m.refreshChecksumAndType(nil, true)
	if err != nil {
		return bundle, err
	}

	if mrn != "" {
		m.Mrn = mrn + "/assetfilter/" + m.CodeId
	}
	m.Title = m.Query
	return bundle, nil
}

// RefreshChecksumAndType by compiling the query and updating the Checksum field
func (m *Mquery) RefreshChecksumAndType(props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	return m.refreshChecksumAndType(props, false)
}

func (m *Mquery) refreshChecksumAndType(props map[string]*llx.Primitive, mustCompileV1 bool) (*llx.CodeBundle, error) {
	bundle, err := m.Compile(props, mustCompileV1)
	if err != nil {
		return bundle, errors.New("failed to compile query '" + m.Query + "': " + err.Error())
	}

	if bundle.GetCodeV2().GetId() == "" {
		return bundle, errors.New("failed to compile query: received empty result values")
	}

	// We think its ok to always use the new code id
	m.CodeId = bundle.CodeV2.Id

	// the compile step also dedents the code
	m.Query = bundle.Source

	// TODO: record multiple entrypoints and types
	// TODO(jaym): is it possible that the 2 could produce different types
	if bundle.DeprecatedV5Code != nil {
		if len(bundle.DeprecatedV5Code.Entrypoints) == 1 {
			ep := bundle.DeprecatedV5Code.Entrypoints[0]
			chunk := bundle.DeprecatedV5Code.Code[ep-1]
			typ := chunk.Type()
			m.Type = string(typ)
		} else {
			m.Type = string(types.Any)
		}
	} else {
		if entrypoints := bundle.CodeV2.Entrypoints(); len(entrypoints) == 1 {
			ep := entrypoints[0]
			chunk := bundle.CodeV2.Chunk(ep)
			typ := chunk.Type()
			m.Type = string(typ)
		} else {
			m.Type = string(types.Any)
		}
	}

	c := checksums.New.
		Add(m.Query).
		Add(m.CodeId).
		Add(bundle.DeprecatedV5Code.GetId()).
		Add(m.Mrn).
		Add(m.Type).
		Add(m.Title).Add("v2")

	if m.Docs != nil {
		c = c.
			Add(m.Docs.Desc).
			Add(m.Docs.Audit).
			Add(m.Docs.Remediation)
	}

	for i := range m.Refs {
		c = c.
			Add(m.Refs[i].Title).
			Add(m.Refs[i].Url)
	}

	keys := make([]string, len(m.Tags))
	i := 0
	for k := range m.Tags {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		c = c.
			Add(k).
			Add(m.Tags[k])
	}

	if m.Severity != nil {
		c = c.AddUint(uint64(m.Severity.Value))
	}

	m.Checksum = c.String()

	return bundle, nil
}

// Sanitize ensure the content is in good shape and removes leading and trailing whitespace
func (m *Mquery) Sanitize() {
	if m == nil {
		return
	}

	if m.Docs != nil {
		m.Docs.Desc = strings.TrimSpace(m.Docs.Desc)
		m.Docs.Audit = strings.TrimSpace(m.Docs.Audit)
		m.Docs.Remediation = strings.TrimSpace(m.Docs.Remediation)
	}

	for i := range m.Refs {
		r := m.Refs[i]
		r.Title = strings.TrimSpace(r.Title)
		r.Url = strings.TrimSpace(r.Url)
	}

	if m.Tags != nil {
		sanitizedTags := map[string]string{}
		for k, v := range m.Tags {
			sk := strings.TrimSpace(k)
			sv := strings.TrimSpace(v)
			sanitizedTags[sk] = sv
		}
		m.Tags = sanitizedTags
	}
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (m *Mquery) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, m.Mrn, "query", m.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", m.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for query "+m.Title)
	}

	m.Mrn = nu
	m.Uid = ""
	return nil
}

func RefreshMRN(ownerMRN string, existingMRN string, resource string, uid string) (string, error) {
	// NOTE: asset policy bundles may not have an owner set, therefore we skip if the query already has an mrn
	if existingMRN != "" {
		if !mrn.IsValid(existingMRN) {
			return "", errors.New("invalid MRN: " + existingMRN)
		}
		return existingMRN, nil
	}

	if ownerMRN == "" {
		return "", errors.New("cannot refresh MRN if the owner MRN is empty")
	}

	if uid == "" {
		return "", errors.New("cannot refresh MRN with an empty UID")
	}

	mrn, err := mrn.NewChildMRN(ownerMRN, resource, uid)
	if err != nil {
		return "", err
	}

	return mrn.String(), nil
}

func ChecksumAssetFilters(queries []*Mquery) (string, error) {
	for i := range queries {
		if _, err := queries[i].refreshChecksumAndType(nil, true); err != nil {
			return "", errors.New("failed to compile query: " + err.Error())
		}
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].CodeId < queries[j].CodeId
	})

	afc := checksums.New
	for i := range queries {
		afc = afc.Add(queries[i].CodeId)
	}

	return afc.String(), nil
}
