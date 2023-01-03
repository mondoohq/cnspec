package bundle

import (
	"gopkg.in/yaml.v3"
)

type FileContext struct {
	Line   int
	Column int
}

// PolicyBundle is a struct only optimized for yaml parsing and formatting. In contrast to the normal k8s yaml parser
// it keeps most of the comments. DO NOT USE THE STRUCT DIRECTLY IN CODE. It is only used for parsing and formatting.
//
// The data structure is copied from policy.Bundle since the yaml.v3 keeps order of fields. Therefore the order
// of the fields matter and allow a nice formatting.
// TODO: figure out how to keep comments and metadata with custom structs
type PolicyBundle struct {
	OwnerMrn    string      `yaml:"owner_mrn,omitempty"`
	Policies    []*Policy   `yaml:"policies,omitempty"`
	Props       []*Mquery   `yaml:"props,omitempty"`
	Queries     []*Mquery   `yaml:"queries,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *PolicyBundle) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias PolicyBundle
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = PolicyBundle(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type AssetFilter struct {
	Query       string      `yaml:"query,omitempty"`
	Indicators  string      `yaml:"indicators,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *AssetFilter) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias AssetFilter
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = AssetFilter(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type AssetFilters struct {
	AssetFilters []*AssetFilter `yaml:"asset_filters,omitempty"`
	FileContext  FileContext    `yaml:"-""`
}

func (p *AssetFilters) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias AssetFilters
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = AssetFilters(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type DataType int32

type Mquery struct {
	Uid         string            `yaml:"uid,omitempty"`
	Title       string            `yaml:"title,omitempty"`
	Severity    int64             `yaml:"severity,omitempty"`
	Checksum    string            `yaml:"checksum,omitempty"`
	Type        DataType          `yaml:"type,omitempty"`
	Docs        *MqueryDocs       `yaml:"docs,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty"`
	Refs        []*MqueryRef      `yaml:"refs,omitempty"`
	Query       string            `yaml:"query,omitempty"`
	FileContext FileContext       `yaml:"-""`
}

func (p *Mquery) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias Mquery
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = Mquery(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type MqueryDocs struct {
	Desc        string      ` yaml:"desc,omitempty"`
	Audit       string      `yaml:"audit,omitempty"`
	Remediation string      `yaml:"remediation,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *MqueryDocs) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias MqueryDocs
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = MqueryDocs(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type MqueryRef struct {
	Title       string      `yaml:"title,omitempty"`
	Url         string      `yaml:"url,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *MqueryRef) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias MqueryRef
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = MqueryRef(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type Author struct {
	Name        string      `yaml:"name,omitempty"`
	Email       string      `yaml:"email,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *Author) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias Author
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = Author(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type PolicyVar struct {
	Name        string      `yaml:"name,omitempty"`
	Query       string      `yaml:"query,omitempty"`
	FileContext FileContext `yaml:"-""`
}

func (p *PolicyVar) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias PolicyVar
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = PolicyVar(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type (
	ScoringSystem int32
	QueryAction   int32
)

type ScoringSpec struct {
	Id                 string        `yaml:"id,omitempty"`
	Weight             uint32        `yaml:"weight,omitempty"`
	WeightIsPercentage bool          `yaml:"weight_is_percentage,omitempty"`
	ScoringSystem      ScoringSystem `yaml:"scoring_system,omitempty"`
	Action             QueryAction   `yaml:"action,omitempty"`
	FileContext        FileContext   `yaml:"-""`
}

func (p *ScoringSpec) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias ScoringSpec
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = ScoringSpec(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type PolicySpec struct {
	Title             string                  `yaml:"title,omitempty"`
	Docs              *PolicySpecDocs         `yaml:"docs,omitempty"`
	AssetFilter       *AssetFilter            `yaml:"asset_filter,omitempty"`
	ExecutionChecksum string                  `yaml:"execution_checksum,omitempty"`
	ScoringChecksum   string                  `yaml:"scoring_checksum,omitempty"`
	StartDate         int64                   `yaml:"start_date,omitempty"`
	EndDate           int64                   `yaml:"end_date,omitempty"`
	ReminderDate      int64                   `yaml:"reminder_date,omitempty"`
	Created           int64                   `yaml:"created,omitempty"`
	Modified          int64                   `yaml:"modified,omitempty"`
	Policies          map[string]*ScoringSpec `yaml:"policies,omitempty"`
	ScoringQueries    map[string]*ScoringSpec `yaml:"scoring_queries,omitempty"`
	DataQueries       map[string]QueryAction  `yaml:"data_queries,omitempty"`
	FileContext       FileContext             `yaml:"-""`
}

func (p *PolicySpec) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias PolicySpec
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = PolicySpec(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

func (ps *PolicySpec) Clone() *PolicySpec {
	if ps == nil {
		return nil
	}
	clone := &PolicySpec{
		Title:             ps.Title,
		ExecutionChecksum: ps.ExecutionChecksum,
		ScoringChecksum:   ps.ScoringChecksum,
		StartDate:         ps.StartDate,
		EndDate:           ps.EndDate,
		ReminderDate:      ps.ReminderDate,
		Created:           ps.Created,
		Modified:          ps.Modified,
		Policies:          ps.Policies,
		ScoringQueries:    ps.ScoringQueries,
		DataQueries:       ps.DataQueries,
	}

	if ps.AssetFilter != nil {
		clone.AssetFilter = &AssetFilter{
			Query:      ps.AssetFilter.Query,
			Indicators: ps.AssetFilter.Indicators,
		}
	}

	return clone
}

type PolicySpecDocs struct {
	Desc string `yaml:"desc,omitempty"`
}

type Policy struct {
	Uid           string                  `yaml:"uid,omitempty"`
	Mrn           string                  `yaml:"mrn,omitempty"`
	Name          string                  `yaml:"name,omitempty"`
	Version       string                  `yaml:"version,omitempty"`
	OwnerMrn      string                  `yaml:"owner_mrn,omitempty"`
	Authors       []*Author               `yaml:"authors,omitempty"`
	Created       int64                   `yaml:"created,omitempty"`
	Modified      int64                   `yaml:"modified,omitempty"`
	IsPublic      bool                    `yaml:"is_public,omitempty"`
	Tags          map[string]string       `yaml:"tags,omitempty"`
	Props         map[string]string       `yaml:"props,omitempty" `
	AssetFilters  map[string]*AssetFilter `yaml:"asset_filters,omitempty"`
	ScoringSystem ScoringSystem           `yaml:"scoring_system,omitempty"`
	Specs         []*PolicySpec           `yaml:"specs,omitempty"`
	Vars          []*PolicyVar            `yaml:"vars,omitempty"`
	Docs          *PolicyDocs             `yaml:"docs,omitempty"`
	FileContext   FileContext             `yaml:"-""`
}

func (p *Policy) UnmarshalYAML(node *yaml.Node) error {
	// need alias object to circumvent the UnmarshalYAML interface
	type alias Policy
	var obj alias
	err := node.Decode(&obj)
	if err != nil {
		return err
	}
	*p = Policy(obj)
	// extract file context from node object
	p.FileContext.Column = node.Column
	p.FileContext.Line = node.Line
	return nil
}

type PolicyDocs struct {
	Desc        string      `yaml:"desc,omitempty"`
	FileContext FileContext `yaml:"-""`
}

// ParseYaml loads a yaml file and parse it into the go struct
func ParseYaml(data []byte) (*PolicyBundle, error) {
	baseline := PolicyBundle{}

	err := yaml.Unmarshal([]byte(data), &baseline)
	if err != nil {
		return nil, err
	}

	return &baseline, nil
}
