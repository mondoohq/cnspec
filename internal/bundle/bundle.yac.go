// Code generated by yac-it. DO NOT EDIT.
//
// Configure yac-it for things you want to auto-generate and extend generated
// objects in a separate file please.

package bundle

import (
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnspec/policy"
	"gopkg.in/yaml.v3"
)

type FileContext struct {
	Line   int
	Column int
}

type MqueryDocs struct {
	Desc        string       `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	Audit       string       `protobuf:"bytes,2,opt,name=audit,proto3" json:"audit,omitempty" yaml:"audit,omitempty"`
	Refs        []*MqueryRef `protobuf:"bytes,4,rep,name=refs,proto3" json:"refs,omitempty" yaml:"refs,omitempty"`
	Remediation *Remediation `protobuf:"bytes,5,opt,name=remediation,proto3" json:"remediation,omitempty" yaml:"remediation,omitempty"`
	FileContext FileContext  `json:"-" yaml:"-"`
}

func (x *MqueryDocs) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp MqueryDocs
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Property struct {
	Mql         string       `protobuf:"bytes,1,opt,name=mql,proto3" json:"mql,omitempty" yaml:"mql,omitempty"`
	CodeId      string       `protobuf:"bytes,2,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id,omitempty"`
	Checksum    string       `protobuf:"bytes,3,opt,name=checksum,proto3" json:"checksum,omitempty" yaml:"checksum,omitempty"`
	Mrn         string       `protobuf:"bytes,4,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid         string       `protobuf:"bytes,5,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Type        string       `protobuf:"bytes,6,opt,name=type,proto3" json:"type,omitempty" yaml:"type,omitempty"`
	Context     string       `protobuf:"bytes,7,opt,name=context,proto3" json:"context,omitempty" yaml:"context,omitempty"`
	For         []*ObjectRef `protobuf:"bytes,8,rep,name=for,proto3" json:"for,omitempty" yaml:"for,omitempty"`
	Title       string       `protobuf:"bytes,20,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Desc        string       `protobuf:"bytes,35,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	FileContext FileContext  `json:"-" yaml:"-"`
}

func (x *Property) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Property
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_Policy struct {
	Mrn                    string                          `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Name                   string                          `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty" yaml:"name,omitempty"`
	Version                string                          `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty" yaml:"version,omitempty"`
	LocalContentChecksum   string                          `protobuf:"bytes,37,opt,name=local_content_checksum,json=localContentChecksum,proto3" json:"local_content_checksum,omitempty" yaml:"local_content_checksum,omitempty"`
	GraphContentChecksum   string                          `protobuf:"bytes,38,opt,name=graph_content_checksum,json=graphContentChecksum,proto3" json:"graph_content_checksum,omitempty" yaml:"graph_content_checksum,omitempty"`
	LocalExecutionChecksum string                          `protobuf:"bytes,39,opt,name=local_execution_checksum,json=localExecutionChecksum,proto3" json:"local_execution_checksum,omitempty" yaml:"local_execution_checksum,omitempty"`
	GraphExecutionChecksum string                          `protobuf:"bytes,40,opt,name=graph_execution_checksum,json=graphExecutionChecksum,proto3" json:"graph_execution_checksum,omitempty" yaml:"graph_execution_checksum,omitempty"`
	Specs                  []*DeprecatedV7_PolicySpec      `protobuf:"bytes,6,rep,name=specs,proto3" json:"specs,omitempty" yaml:"specs,omitempty"`
	AssetFilters           map[string]*DeprecatedV7_Mquery `protobuf:"bytes,7,rep,name=asset_filters,json=assetFilters,proto3" json:"asset_filters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"asset_filters,omitempty"`
	OwnerMrn               string                          `protobuf:"bytes,8,opt,name=owner_mrn,json=ownerMrn,proto3" json:"owner_mrn,omitempty" yaml:"owner_mrn,omitempty"`
	IsPublic               bool                            `protobuf:"varint,9,opt,name=is_public,json=isPublic,proto3" json:"is_public,omitempty" yaml:"is_public,omitempty"`
	ScoringSystem          policy.ScoringSystem            `protobuf:"varint,10,opt,name=scoring_system,json=scoringSystem,proto3,enum=cnspec.policy.v1.ScoringSystem" json:"scoring_system,omitempty" yaml:"scoring_system,omitempty"`
	Authors                []*DeprecatedV7_Author          `protobuf:"bytes,30,rep,name=authors,proto3" json:"authors,omitempty" yaml:"authors,omitempty"`
	Created                int64                           `protobuf:"varint,32,opt,name=created,proto3" json:"created,omitempty" yaml:"created,omitempty"`
	Modified               int64                           `protobuf:"varint,33,opt,name=modified,proto3" json:"modified,omitempty" yaml:"modified,omitempty"`
	Tags                   map[string]string               `protobuf:"bytes,34,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"tags,omitempty"`
	Props                  map[string]string               `protobuf:"bytes,35,rep,name=props,proto3" json:"props,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"props,omitempty"`
	Uid                    string                          `protobuf:"bytes,36,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Docs                   *PolicyDocs                     `protobuf:"bytes,41,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	QueryCounts            *QueryCounts                    `protobuf:"bytes,42,opt,name=query_counts,json=queryCounts,proto3" json:"query_counts,omitempty" yaml:"query_counts,omitempty"`
	FileContext            FileContext                     `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_Policy) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_Policy
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_ScoringSpec struct {
	Id                 string                      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" yaml:"id,omitempty"`
	Weight             uint32                      `protobuf:"varint,2,opt,name=weight,proto3" json:"weight,omitempty" yaml:"weight,omitempty"`
	WeightIsPercentage bool                        `protobuf:"varint,3,opt,name=weight_is_percentage,json=weightIsPercentage,proto3" json:"weight_is_percentage,omitempty" yaml:"weight_is_percentage,omitempty"`
	ScoringSystem      policy.ScoringSystem        `protobuf:"varint,4,opt,name=scoring_system,json=scoringSystem,proto3,enum=cnspec.policy.v1.ScoringSystem" json:"scoring_system,omitempty" yaml:"scoring_system,omitempty"`
	Action             policy.QueryAction          `protobuf:"varint,6,opt,name=action,proto3,enum=cnspec.policy.v1.QueryAction" json:"action,omitempty" yaml:"action,omitempty"`
	Severity           *DeprecatedV7_SeverityValue `protobuf:"bytes,7,opt,name=severity,proto3" json:"severity,omitempty" yaml:"severity,omitempty"`
	FileContext        FileContext                 `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_ScoringSpec) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_ScoringSpec
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Filters struct {
	Items       map[string]*Mquery `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"items,omitempty"`
	FileContext FileContext        `json:"-" yaml:"-"`
}

func (x *Filters) addFileContext(node *yaml.Node) {
	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
}

type DeprecatedV7_Author struct {
	Name        string      `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty" yaml:"name,omitempty"`
	Email       string      `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty" yaml:"email,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_Author) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_Author
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_Mquery struct {
	Query       string                      `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty" yaml:"query,omitempty"`
	CodeId      string                      `protobuf:"bytes,2,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id,omitempty"`
	Checksum    string                      `protobuf:"bytes,3,opt,name=checksum,proto3" json:"checksum,omitempty" yaml:"checksum,omitempty"`
	Mrn         string                      `protobuf:"bytes,4,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid         string                      `protobuf:"bytes,5,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Type        string                      `protobuf:"bytes,6,opt,name=type,proto3" json:"type,omitempty" yaml:"type,omitempty"`
	Severity    *DeprecatedV7_SeverityValue `protobuf:"bytes,19,opt,name=severity,proto3" json:"severity,omitempty" yaml:"severity,omitempty"`
	Title       string                      `protobuf:"bytes,20,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Docs        *DeprecatedV7_MqueryDocs    `protobuf:"bytes,21,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	Refs        []*DeprecatedV7_MqueryRef   `protobuf:"bytes,22,rep,name=refs,proto3" json:"refs,omitempty" yaml:"refs,omitempty"`
	Tags        map[string]string           `protobuf:"bytes,34,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"tags,omitempty"`
	FileContext FileContext                 `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_Mquery) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_Mquery
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type PolicyGroupDocs struct {
	Desc        string      `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *PolicyGroupDocs) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp PolicyGroupDocs
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type PolicyRef struct {
	Mrn         string          `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid         string          `protobuf:"bytes,2,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Action      explorer.Action `protobuf:"varint,41,opt,name=action,proto3,enum=cnquery.explorer.Action" json:"action,omitempty" yaml:"action,omitempty"`
	Impact      *Impact         `protobuf:"bytes,23,opt,name=impact,proto3" json:"impact,omitempty" yaml:"impact,omitempty"`
	Checksum    string          `protobuf:"bytes,4,opt,name=checksum,proto3" json:"checksum,omitempty" yaml:"checksum,omitempty"`
	FileContext FileContext     `json:"-" yaml:"-"`
}

func (x *PolicyRef) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp PolicyRef
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Remediation struct {
	Items       []*TypedDoc `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty" yaml:"items,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *Remediation) addFileContext(node *yaml.Node) {
	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
}

type DeprecatedV7_PolicySpec struct {
	Policies       map[string]*DeprecatedV7_ScoringSpec `protobuf:"bytes,1,rep,name=policies,proto3" json:"policies,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"policies,omitempty"`
	ScoringQueries map[string]*DeprecatedV7_ScoringSpec `protobuf:"bytes,2,rep,name=scoring_queries,json=scoringQueries,proto3" json:"scoring_queries,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"scoring_queries,omitempty"`
	DataQueries    map[string]policy.QueryAction        `protobuf:"bytes,3,rep,name=data_queries,json=dataQueries,proto3" json:"data_queries,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3,enum=cnspec.policy.v1.QueryAction" yaml:"data_queries,omitempty"`
	AssetFilter    *DeprecatedV7_Mquery                 `protobuf:"bytes,20,opt,name=asset_filter,json=assetFilter,proto3" json:"asset_filter,omitempty" yaml:"asset_filter,omitempty"`
	StartDate      int64                                `protobuf:"varint,21,opt,name=start_date,json=startDate,proto3" json:"start_date,omitempty" yaml:"start_date,omitempty"`
	EndDate        int64                                `protobuf:"varint,22,opt,name=end_date,json=endDate,proto3" json:"end_date,omitempty" yaml:"end_date,omitempty"`
	ReminderDate   int64                                `protobuf:"varint,23,opt,name=reminder_date,json=reminderDate,proto3" json:"reminder_date,omitempty" yaml:"reminder_date,omitempty"`
	Title          string                               `protobuf:"bytes,24,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Docs           *PolicyGroupDocs                     `protobuf:"bytes,25,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	Created        int64                                `protobuf:"varint,32,opt,name=created,proto3" json:"created,omitempty" yaml:"created,omitempty"`
	Modified       int64                                `protobuf:"varint,33,opt,name=modified,proto3" json:"modified,omitempty" yaml:"modified,omitempty"`
	FileContext    FileContext                          `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_PolicySpec) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_PolicySpec
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_MqueryDocs struct {
	Desc        string      `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	Audit       string      `protobuf:"bytes,2,opt,name=audit,proto3" json:"audit,omitempty" yaml:"audit,omitempty"`
	Remediation string      `protobuf:"bytes,3,opt,name=remediation,proto3" json:"remediation,omitempty" yaml:"remediation,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_MqueryDocs) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_MqueryDocs
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type QueryCounts struct {
	ScoringCount int64       `protobuf:"varint,1,opt,name=scoring_count,json=scoringCount,proto3" json:"scoring_count,omitempty" yaml:"scoring_count,omitempty"`
	DataCount    int64       `protobuf:"varint,2,opt,name=data_count,json=dataCount,proto3" json:"data_count,omitempty" yaml:"data_count,omitempty"`
	TotalCount   int64       `protobuf:"varint,3,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty" yaml:"total_count,omitempty"`
	FileContext  FileContext `json:"-" yaml:"-"`
}

func (x *QueryCounts) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp QueryCounts
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type ImpactValue struct {
	Value       int32       `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty" yaml:"value,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *ImpactValue) addFileContext(node *yaml.Node) {
	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
}

type MqueryRef struct {
	Title       string      `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Url         string      `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty" yaml:"url,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *MqueryRef) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp MqueryRef
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Author struct {
	Name        string      `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty" yaml:"name,omitempty"`
	Email       string      `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty" yaml:"email,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *Author) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Author
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Bundle struct {
	OwnerMrn             string                 `protobuf:"bytes,1,opt,name=owner_mrn,json=ownerMrn,proto3" json:"owner_mrn,omitempty" yaml:"owner_mrn,omitempty"`
	Policies             []*Policy              `protobuf:"bytes,7,rep,name=policies,proto3" json:"policies,omitempty" yaml:"policies,omitempty"`
	Props                []*Property            `protobuf:"bytes,3,rep,name=props,proto3" json:"props,omitempty" yaml:"props,omitempty"`
	Queries              []*Mquery              `protobuf:"bytes,6,rep,name=queries,proto3" json:"queries,omitempty" yaml:"queries,omitempty"`
	Docs                 *PolicyDocs            `protobuf:"bytes,5,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	DeprecatedV7Policies []*DeprecatedV7_Policy `protobuf:"bytes,2,rep,name=deprecated_v7_policies,json=deprecatedV7Policies,proto3" json:"deprecated_v7_policies,omitempty" yaml:"deprecated_v7_policies,omitempty"`
	DeprecatedV7Queries  []*DeprecatedV7_Mquery `protobuf:"bytes,4,rep,name=deprecated_v7_queries,json=deprecatedV7Queries,proto3" json:"deprecated_v7_queries,omitempty" yaml:"deprecated_v7_queries,omitempty"`
	FileContext          FileContext            `json:"-" yaml:"-"`
}

func (x *Bundle) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Bundle
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Policy struct {
	Specs                  []*DeprecatedV7_PolicySpec      `protobuf:"bytes,6,rep,name=specs,proto3" json:"specs,omitempty" yaml:"specs,omitempty"`
	AssetFilters           map[string]*DeprecatedV7_Mquery `protobuf:"bytes,7,rep,name=asset_filters,json=assetFilters,proto3" json:"asset_filters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"asset_filters,omitempty"`
	Mrn                    string                          `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid                    string                          `protobuf:"bytes,36,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Name                   string                          `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty" yaml:"name,omitempty"`
	Version                string                          `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty" yaml:"version,omitempty"`
	OwnerMrn               string                          `protobuf:"bytes,8,opt,name=owner_mrn,json=ownerMrn,proto3" json:"owner_mrn,omitempty" yaml:"owner_mrn,omitempty"`
	Groups                 []*PolicyGroup                  `protobuf:"bytes,11,rep,name=groups,proto3" json:"groups,omitempty" yaml:"groups,omitempty"`
	License                string                          `protobuf:"bytes,21,opt,name=license,proto3" json:"license,omitempty" yaml:"license,omitempty"`
	Docs                   *PolicyDocs                     `protobuf:"bytes,41,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	Summary                string                          `protobuf:"bytes,46,opt,name=summary,proto3" json:"summary,omitempty" yaml:"summary,omitempty"`
	ScoringSystem          policy.ScoringSystem            `protobuf:"varint,10,opt,name=scoring_system,json=scoringSystem,proto3,enum=cnspec.policy.v1.ScoringSystem" json:"scoring_system,omitempty" yaml:"scoring_system,omitempty"`
	Authors                []*Author                       `protobuf:"bytes,30,rep,name=authors,proto3" json:"authors,omitempty" yaml:"authors,omitempty"`
	Created                int64                           `protobuf:"varint,32,opt,name=created,proto3" json:"created,omitempty" yaml:"created,omitempty"`
	Modified               int64                           `protobuf:"varint,33,opt,name=modified,proto3" json:"modified,omitempty" yaml:"modified,omitempty"`
	Tags                   map[string]string               `protobuf:"bytes,34,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"tags,omitempty"`
	Props                  []*Property                     `protobuf:"bytes,45,rep,name=props,proto3" json:"props,omitempty" yaml:"props,omitempty"`
	LocalContentChecksum   string                          `protobuf:"bytes,37,opt,name=local_content_checksum,json=localContentChecksum,proto3" json:"local_content_checksum,omitempty" yaml:"local_content_checksum,omitempty"`
	GraphContentChecksum   string                          `protobuf:"bytes,38,opt,name=graph_content_checksum,json=graphContentChecksum,proto3" json:"graph_content_checksum,omitempty" yaml:"graph_content_checksum,omitempty"`
	LocalExecutionChecksum string                          `protobuf:"bytes,39,opt,name=local_execution_checksum,json=localExecutionChecksum,proto3" json:"local_execution_checksum,omitempty" yaml:"local_execution_checksum,omitempty"`
	GraphExecutionChecksum string                          `protobuf:"bytes,40,opt,name=graph_execution_checksum,json=graphExecutionChecksum,proto3" json:"graph_execution_checksum,omitempty" yaml:"graph_execution_checksum,omitempty"`
	ComputedFilters        *Filters                        `protobuf:"bytes,43,opt,name=computed_filters,json=computedFilters,proto3" json:"computed_filters,omitempty" yaml:"computed_filters,omitempty"`
	QueryCounts            *QueryCounts                    `protobuf:"bytes,42,opt,name=query_counts,json=queryCounts,proto3" json:"query_counts,omitempty" yaml:"query_counts,omitempty"`
	FileContext            FileContext                     `json:"-" yaml:"-"`
}

func (x *Policy) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Policy
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Mquery struct {
	Query       string            `protobuf:"bytes,40,opt,name=query,proto3" json:"query,omitempty" yaml:"query,omitempty"`
	Refs        []*MqueryRef      `protobuf:"bytes,22,rep,name=refs,proto3" json:"refs,omitempty" yaml:"refs,omitempty"`
	Mql         string            `protobuf:"bytes,1,opt,name=mql,proto3" json:"mql,omitempty" yaml:"mql,omitempty"`
	CodeId      string            `protobuf:"bytes,2,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id,omitempty"`
	Checksum    string            `protobuf:"bytes,3,opt,name=checksum,proto3" json:"checksum,omitempty" yaml:"checksum,omitempty"`
	Mrn         string            `protobuf:"bytes,4,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid         string            `protobuf:"bytes,5,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	Type        string            `protobuf:"bytes,6,opt,name=type,proto3" json:"type,omitempty" yaml:"type,omitempty"`
	Context     string            `protobuf:"bytes,7,opt,name=context,proto3" json:"context,omitempty" yaml:"context,omitempty"`
	Title       string            `protobuf:"bytes,20,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Docs        *MqueryDocs       `protobuf:"bytes,21,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	Desc        string            `protobuf:"bytes,35,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	Impact      *Impact           `protobuf:"bytes,23,opt,name=impact,proto3" json:"impact,omitempty" yaml:"impact,omitempty"`
	Tags        map[string]string `protobuf:"bytes,34,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3" yaml:"tags,omitempty"`
	Filters     *Filters          `protobuf:"bytes,37,opt,name=filters,proto3" json:"filters,omitempty" yaml:"filters,omitempty"`
	Props       []*Property       `protobuf:"bytes,38,rep,name=props,proto3" json:"props,omitempty" yaml:"props,omitempty"`
	Variants    []*Mquery         `protobuf:"bytes,39,rep,name=variants,proto3" json:"variants,omitempty" yaml:"variants,omitempty"`
	Action      explorer.Action   `protobuf:"varint,41,opt,name=action,proto3,enum=cnquery.explorer.Action" json:"action,omitempty" yaml:"action,omitempty"`
	FileContext FileContext       `json:"-" yaml:"-"`
}

func (x *Mquery) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp Mquery
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type PolicyDocs struct {
	Desc        string      `protobuf:"bytes,1,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *PolicyDocs) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp PolicyDocs
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_Bundle struct {
	OwnerMrn    string                 `protobuf:"bytes,1,opt,name=owner_mrn,json=ownerMrn,proto3" json:"owner_mrn,omitempty" yaml:"owner_mrn,omitempty"`
	Queries     []*DeprecatedV7_Mquery `protobuf:"bytes,4,rep,name=queries,proto3" json:"queries,omitempty" yaml:"queries,omitempty"`
	Policies    []*DeprecatedV7_Policy `protobuf:"bytes,2,rep,name=policies,proto3" json:"policies,omitempty" yaml:"policies,omitempty"`
	Props       []*DeprecatedV7_Mquery `protobuf:"bytes,3,rep,name=props,proto3" json:"props,omitempty" yaml:"props,omitempty"`
	Docs        *PolicyDocs            `protobuf:"bytes,5,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	FileContext FileContext            `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_Bundle) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_Bundle
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_MqueryRef struct {
	Title       string      `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Url         string      `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty" yaml:"url,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_MqueryRef) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_MqueryRef
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type Impact struct {
	Value       *ImpactValue                  `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty" yaml:"value,omitempty"`
	Scoring     explorer.Impact_ScoringSystem `protobuf:"varint,2,opt,name=scoring,proto3,enum=cnquery.explorer.Impact_ScoringSystem" json:"scoring,omitempty" yaml:"scoring,omitempty"`
	Weight      int32                         `protobuf:"varint,3,opt,name=weight,proto3" json:"weight,omitempty" yaml:"weight,omitempty"`
	Action      explorer.Action               `protobuf:"varint,4,opt,name=action,proto3,enum=cnquery.explorer.Action" json:"action,omitempty" yaml:"action,omitempty"`
	FileContext FileContext                   `json:"-" yaml:"-"`
}

func (x *Impact) addFileContext(node *yaml.Node) {
	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
}

type TypedDoc struct {
	Id          string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" yaml:"id,omitempty"`
	Desc        string      `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty" yaml:"desc,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *TypedDoc) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp TypedDoc
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type ObjectRef struct {
	Mrn         string      `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty" yaml:"mrn,omitempty"`
	Uid         string      `protobuf:"bytes,2,opt,name=uid,proto3" json:"uid,omitempty" yaml:"uid,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *ObjectRef) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp ObjectRef
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type DeprecatedV7_SeverityValue struct {
	Value       int64       `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty" yaml:"value,omitempty"`
	FileContext FileContext `json:"-" yaml:"-"`
}

func (x *DeprecatedV7_SeverityValue) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp DeprecatedV7_SeverityValue
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}

type PolicyGroup struct {
	Policies     []*PolicyRef     `protobuf:"bytes,1,rep,name=policies,proto3" json:"policies,omitempty" yaml:"policies,omitempty"`
	Checks       []*Mquery        `protobuf:"bytes,2,rep,name=checks,proto3" json:"checks,omitempty" yaml:"checks,omitempty"`
	Queries      []*Mquery        `protobuf:"bytes,3,rep,name=queries,proto3" json:"queries,omitempty" yaml:"queries,omitempty"`
	Type         policy.GroupType `protobuf:"varint,4,opt,name=type,proto3,enum=cnspec.policy.v1.GroupType" json:"type,omitempty" yaml:"type,omitempty"`
	Filters      *Filters         `protobuf:"bytes,20,opt,name=filters,proto3" json:"filters,omitempty" yaml:"filters,omitempty"`
	StartDate    int64            `protobuf:"varint,21,opt,name=start_date,json=startDate,proto3" json:"start_date,omitempty" yaml:"start_date,omitempty"`
	EndDate      int64            `protobuf:"varint,22,opt,name=end_date,json=endDate,proto3" json:"end_date,omitempty" yaml:"end_date,omitempty"`
	ReminderDate int64            `protobuf:"varint,23,opt,name=reminder_date,json=reminderDate,proto3" json:"reminder_date,omitempty" yaml:"reminder_date,omitempty"`
	Title        string           `protobuf:"bytes,24,opt,name=title,proto3" json:"title,omitempty" yaml:"title,omitempty"`
	Docs         *PolicyGroupDocs `protobuf:"bytes,25,opt,name=docs,proto3" json:"docs,omitempty" yaml:"docs,omitempty"`
	Created      int64            `protobuf:"varint,32,opt,name=created,proto3" json:"created,omitempty" yaml:"created,omitempty"`
	Modified     int64            `protobuf:"varint,33,opt,name=modified,proto3" json:"modified,omitempty" yaml:"modified,omitempty"`
	FileContext  FileContext      `json:"-" yaml:"-"`
}

func (x *PolicyGroup) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp PolicyGroup
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}
