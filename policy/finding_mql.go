// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mqlFinding mirrors the JSON output of the MQL `finding` resource.
// Note: Source, Details, and Affects come through as unexpanded resource
// reference strings (e.g. "finding.source id = ...") rather than objects,
// so they are typed as `any` and skipped during conversion for now.
type mqlFinding struct {
	Id           string           `json:"id"`
	Ref          string           `json:"ref"`
	Mrn          string           `json:"mrn"`
	GroupId      string           `json:"groupId"`
	Summary      string           `json:"summary"`
	Status       string           `json:"status"`
	FirstSeenAt  *string          `json:"firstSeenAt"`
	LastSeenAt   *string          `json:"lastSeenAt"`
	RemediatedAt *string          `json:"remediatedAt"`
	Source       any              `json:"source"`
	Details      any              `json:"details"`
	Affects      any              `json:"affects"`
	Evidences    []mqlEvidence    `json:"evidences"`
	Remediations []map[string]any `json:"remediations"`
}

type mqlEvidence struct {
	Confidence  string               `json:"confidence"`
	Properties  map[string]any       `json:"properties"`
	Tactic      *mqlAttackTactic     `json:"tactic"`
	Technique   *mqlAttackTechnique  `json:"technique"`
	User        *mqlFindingUser      `json:"user"`
	File        *mqlFindingFile      `json:"file"`
	Process     *mqlFindingProcess   `json:"process"`
	Container   *mqlFindingContainer `json:"container"`
	Kubernetes  *mqlFindingK8s       `json:"kubernetes"`
	RegistryKey *mqlFindingRegKey    `json:"registryKey"`
	Connection  *mqlFindingConn      `json:"connection"`
}

type mqlAttackTactic struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type mqlAttackTechnique struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type mqlFindingUser struct {
	Id         string         `json:"id"`
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties"`
}

type mqlFindingFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Md5      string `json:"md5"`
	Sha256   string `json:"sha256"`
	Contents string `json:"contents"`
}

type mqlFindingProcess struct {
	Cmdline string             `json:"cmdline"`
	Pid     int64              `json:"pid"`
	Binary  *mqlFindingFile    `json:"binary"`
	Script  *mqlFindingFile    `json:"script"`
	User    *mqlFindingUser    `json:"user"`
	Parent  *mqlFindingProcess `json:"parent"`
}

type mqlFindingContainer struct {
	Name     string `json:"name"`
	ImageUri string `json:"imageUri"`
	Digest   string `json:"digest"`
}

type mqlFindingK8s struct {
	Pods  []mqlK8sPod  `json:"pods"`
	Nodes []mqlK8sNode `json:"nodes"`
}

type mqlK8sPod struct {
	Name       string                `json:"name"`
	Namespace  string                `json:"namespace"`
	Containers []mqlFindingContainer `json:"containers"`
}

type mqlK8sNode struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type mqlFindingRegKey struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Data string `json:"data"`
}

type mqlFindingConn struct {
	DestinationAddress string `json:"destinationAddress"`
	DestinationPort    int32  `json:"destinationPort"`
	SourceAddress      string `json:"sourceAddress"`
	SourcePort         int32  `json:"sourcePort"`
	Protocol           string `json:"protocol"`
}

// findingResourceName is the MQL resource type name for findings.
const findingResourceName = "finding"

// IsFindingType checks if the given type represents a finding resource
// or an array of finding resources.
func IsFindingType(typ types.Type) bool {
	if typ.NotSet() {
		return false
	}
	if typ.IsResource() && typ.ResourceName() == findingResourceName {
		return true
	}
	if typ.IsArray() {
		child := typ.Child()
		if child.IsResource() && child.ResourceName() == findingResourceName {
			return true
		}
	}
	return false
}

// BuildCodeBundleMap creates a reverse map from data point checksums to their
// parent code bundle. This allows looking up the code bundle for a given
// RawResult.CodeID.
func BuildCodeBundleMap(queries map[string]*ExecutionQuery) map[string]*llx.CodeBundle {
	m := make(map[string]*llx.CodeBundle)
	for _, eq := range queries {
		if eq.Code == nil || eq.Code.CodeV2 == nil {
			continue
		}
		cb := eq.Code
		// Map entrypoint checksums
		if len(cb.CodeV2.Blocks) > 0 {
			for _, ref := range cb.CodeV2.Blocks[0].Entrypoints {
				if checksum, ok := cb.CodeV2.Checksums[ref]; ok {
					m[checksum] = cb
				}
			}
			// Map datapoint checksums
			for _, ref := range cb.CodeV2.Blocks[0].Datapoints {
				if checksum, ok := cb.CodeV2.Checksums[ref]; ok {
					m[checksum] = cb
				}
			}
		}
	}
	return m
}

// ExtractFindings examines raw results for finding resources and converts them
// to FindingDocument protos. It follows the JSON intermediary pattern: serialize
func ExtractFindings(results []*llx.RawResult, codeBundleMap map[string]*llx.CodeBundle) []*FindingDocument {
	// RawData to JSON, unmarshal into Go structs, then convert to proto.
	var findings []*FindingDocument

	for _, rr := range results {
		if rr.Data == nil || rr.Data.Value == nil || rr.Data.Error != nil {
			continue
		}

		cb, ok := codeBundleMap[rr.CodeID]
		if !ok {
			continue
		}

		jsonBytes := rr.Data.JSON(rr.CodeID, cb)
		if len(jsonBytes) == 0 {
			continue
		}

		extracted := parseFindingsFromJSON(jsonBytes)
		findings = append(findings, extracted...)
	}

	return findings
}

// parseFindingsFromJSON tries to unmarshal JSON bytes as a list of findings.
// It handles both array and single-object formats.
func parseFindingsFromJSON(data []byte) []*FindingDocument {
	// Try as array of findings first (most common case)
	var findings []mqlFinding
	if err := json.Unmarshal(data, &findings); err == nil {
		return convertFindings(findings)
	}

	// Try as a single finding
	var single mqlFinding
	if err := json.Unmarshal(data, &single); err == nil && single.Id != "" {
		return convertFindings([]mqlFinding{single})
	}

	return nil
}

func convertFindings(findings []mqlFinding) []*FindingDocument {
	var docs []*FindingDocument
	for i := range findings {
		if findings[i].Id == "" {
			continue
		}
		fex := mqlFindingToProto(&findings[i])
		if fex == nil {
			continue
		}
		docs = append(docs, &FindingDocument{
			Finding: &FindingDocument_Fex{
				Fex: fex,
			},
		})
	}
	return docs
}

func mqlFindingToProto(f *mqlFinding) *FindingExchange {
	if f == nil {
		return nil
	}

	result := &FindingExchange{
		Id:      f.Id,
		Ref:     f.Ref,
		Mrn:     f.Mrn,
		GroupId: f.GroupId,
		Summary: f.Summary,
		// Status:  parseFindingStatus(f.Status),
		Status: FindingStatus_FINDING_STATUS_AFFECTED,
		Source: &FindingSource{
			Name: "github-dependabot",
		},
		Details: &FindingDetail{
			Severity: &FindingSeverity{Rating: SeverityRating_SEVERITY_RATING_HIGH},
			Category: FindingDetail_CATEGORY_SECURITY, // hardcoded for now
		},
	}

	if f.FirstSeenAt != nil {
		if t, err := time.Parse(time.RFC3339, *f.FirstSeenAt); err == nil {
			result.FirstSeenAt = timestamppb.New(t)
		}
	}
	if f.LastSeenAt != nil {
		if t, err := time.Parse(time.RFC3339, *f.LastSeenAt); err == nil {
			result.LastSeenAt = timestamppb.New(t)
		}
	}
	if f.RemediatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *f.RemediatedAt); err == nil {
			result.RemediatedAt = timestamppb.New(t)
		}
	}

	// Source, Details, and Affects come through as unexpanded resource reference
	// strings (e.g. "finding.source id = ...") and are skipped for now.

	for _, e := range f.Evidences {
		result.Evidences = append(result.Evidences, convertMqlEvidence(&e))
	}

	for _, m := range f.Remediations {
		result.Remediations = append(result.Remediations, convertRemediation(m))
	}

	return result
}

func convertMqlEvidence(e *mqlEvidence) *FindingEvidence {
	result := &FindingEvidence{
		Confidence: parseConfidence(e.Confidence),
		Properties: toStringMap(e.Properties),
	}

	if e.Tactic != nil {
		result.Tactic = &AttackTactic{
			Id:          e.Tactic.Id,
			Name:        e.Tactic.Name,
			Description: e.Tactic.Description,
		}
	}
	if e.Technique != nil {
		result.Technique = &AttackTechnique{
			Id:          e.Technique.Id,
			Name:        e.Technique.Name,
			Description: e.Technique.Description,
		}
	}

	switch {
	case e.User != nil:
		result.Details = &FindingEvidence_User{
			User: &FindingUser{
				Id:         e.User.Id,
				Name:       e.User.Name,
				Properties: toStringMap(e.User.Properties),
			},
		}
	case e.File != nil:
		result.Details = &FindingEvidence_File{
			File: &FindingFile{
				Path:     e.File.Path,
				Size:     e.File.Size,
				Md5:      e.File.Md5,
				Sha256:   e.File.Sha256,
				Contents: e.File.Contents,
			},
		}
	case e.Process != nil:
		result.Details = &FindingEvidence_Process{
			Process: convertMqlProcess(e.Process),
		}
	case e.Container != nil:
		result.Details = &FindingEvidence_Container{
			Container: &FindingContainer{
				Name:     e.Container.Name,
				ImageUri: e.Container.ImageUri,
				Digest:   e.Container.Digest,
			},
		}
	case e.Kubernetes != nil:
		result.Details = &FindingEvidence_Kubernetes{
			Kubernetes: convertMqlK8s(e.Kubernetes),
		}
	case e.RegistryKey != nil:
		result.Details = &FindingEvidence_RegistryKey{
			RegistryKey: &FindingRegistryKey{
				Path: e.RegistryKey.Path,
				Name: e.RegistryKey.Name,
				Data: e.RegistryKey.Data,
			},
		}
	case e.Connection != nil:
		result.Details = &FindingEvidence_Connection{
			Connection: &FindingConnection{
				DestinationAddress: e.Connection.DestinationAddress,
				DestinationPort:    e.Connection.DestinationPort,
				SourceAddress:      e.Connection.SourceAddress,
				SourcePort:         e.Connection.SourcePort,
				Protocol:           parseConnectionProtocol(e.Connection.Protocol),
			},
		}
	}

	return result
}

func convertMqlProcess(p *mqlFindingProcess) *FindingProcess {
	result := &FindingProcess{
		Cmdline: p.Cmdline,
		Pid:     p.Pid,
	}
	if p.Binary != nil {
		result.Binary = &FindingFile{
			Path: p.Binary.Path, Size: p.Binary.Size,
			Md5: p.Binary.Md5, Sha256: p.Binary.Sha256, Contents: p.Binary.Contents,
		}
	}
	if p.Script != nil {
		result.Script = &FindingFile{
			Path: p.Script.Path, Size: p.Script.Size,
			Md5: p.Script.Md5, Sha256: p.Script.Sha256, Contents: p.Script.Contents,
		}
	}
	if p.User != nil {
		result.User = &FindingUser{
			Id: p.User.Id, Name: p.User.Name,
			Properties: toStringMap(p.User.Properties),
		}
	}
	if p.Parent != nil {
		result.Parent = convertMqlProcess(p.Parent)
	}
	return result
}

func convertMqlK8s(k *mqlFindingK8s) *FindingKubernetes {
	result := &FindingKubernetes{}
	for _, pod := range k.Pods {
		p := &FindingKubernetes_Pod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}
		for _, c := range pod.Containers {
			p.Containers = append(p.Containers, &FindingContainer{
				Name: c.Name, ImageUri: c.ImageUri, Digest: c.Digest,
			})
		}
		result.Pods = append(result.Pods, p)
	}
	for _, node := range k.Nodes {
		result.Nodes = append(result.Nodes, &FindingKubernetes_Node{
			Name: node.Name,
			Id:   node.Id,
		})
	}
	return result
}

func convertRemediation(m map[string]any) *FindingRemediation {
	r := &FindingRemediation{}
	if v, ok := m["category"].(string); ok {
		r.Category = parseRemediationCategory(v)
	}
	if v, ok := m["summary"].(string); ok {
		r.Summary = v
	}
	if v, ok := m["details"].(string); ok {
		r.Details = v
	}
	if v, ok := m["fix_type"].(string); ok {
		r.FixType = v
	}
	if v, ok := m["fixType"].(string); ok && r.FixType == "" {
		r.FixType = v
	}
	if v, ok := m["fix_id"].(string); ok {
		r.FixId = v
	}
	if v, ok := m["fixId"].(string); ok && r.FixId == "" {
		r.FixId = v
	}
	if v, ok := m["url"].(string); ok {
		r.Url = v
	}
	return r
}

// Enum parsing helpers

func parseFindingStatus(s string) FindingStatus {
	switch strings.ToLower(s) {
	case "not_affected", "not affected", "notaffected":
		return FindingStatus_FINDING_STATUS_NOT_AFFECTED
	case "affected", "open":
		return FindingStatus_FINDING_STATUS_AFFECTED
	case "fixed":
		return FindingStatus_FINDING_STATUS_FIXED
	case "under_investigation", "under investigation", "underinvestigation":
		return FindingStatus_FINDING_STATUS_UNDER_INVESTIGATION
	case "false_positive", "false positive", "falsepositive":
		return FindingStatus_FINDING_STATUS_FALSE_POSITIVE
	case "wont_fix", "wont fix", "wontfix":
		return FindingStatus_FINDING_STATUS_WONT_FIX
	default:
		return FindingStatus_FINDING_STATUS_UNSPECIFIED
	}
}

func parseConfidence(s string) Confidence {
	switch strings.ToLower(s) {
	case "low":
		return Confidence_CONFIDENCE_LOW
	case "medium":
		return Confidence_CONFIDENCE_MEDIUM
	case "high":
		return Confidence_CONFIDENCE_HIGH
	default:
		return Confidence_CONFIDENCE_UNSPECIFIED
	}
}

func parseConnectionProtocol(s string) FindingConnection_ConnectionProtocol {
	switch strings.ToLower(s) {
	case "icmp":
		return FindingConnection_ICMP
	case "tcp":
		return FindingConnection_TCP
	case "udp":
		return FindingConnection_UDP
	default:
		return FindingConnection_UNSPECIFIED
	}
}

func parseRemediationCategory(s string) FindingRemediation_Category {
	switch strings.ToLower(s) {
	case "no_fix_planned", "nofixplanned":
		return FindingRemediation_NoFixPlanned
	case "none_available", "noneavailable":
		return FindingRemediation_NoneAvailable
	case "fix":
		return FindingRemediation_Fix
	case "workaround":
		return FindingRemediation_Workaround
	default:
		return FindingRemediation_Unspecified
	}
}

// toStringMap converts a map[string]any to map[string]string.
func toStringMap(m map[string]any) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
