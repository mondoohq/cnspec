// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

// // MqlFindingToProto converts an MQL finding resource to a proto FindingExchange.
// func MqlFindingToProto(f *resources.MqlFinding) (*FindingExchange, error) {
// 	if f == nil {
// 		return nil, nil
// 	}

// 	result := &FindingExchange{
// 		Id:      f.Id.Data,
// 		Ref:     f.Ref.Data,
// 		Mrn:     f.Mrn.Data,
// 		GroupId: f.GroupId.Data,
// 		Summary: f.Summary.Data,
// 		Status:  parseFindingStatus(f.Status.Data),
// 	}

// 	if f.FirstSeenAt.Data != nil {
// 		result.FirstSeenAt = timestamppb.New(*f.FirstSeenAt.Data)
// 	}
// 	if f.LastSeenAt.Data != nil {
// 		result.LastSeenAt = timestamppb.New(*f.LastSeenAt.Data)
// 	}
// 	if f.RemediatedAt.Data != nil {
// 		result.RemediatedAt = timestamppb.New(*f.RemediatedAt.Data)
// 	}

// 	if f.Source.Data != nil {
// 		result.Source = convertFindingSource(f.Source.Data)
// 	}

// 	if f.Details.Data != nil {
// 		result.Details = convertFindingDetail(f.Details.Data)
// 	}

// 	for _, item := range f.Affects.Data {
// 		a, ok := item.(*resources.MqlFindingAffects)
// 		if !ok {
// 			continue
// 		}
// 		result.Affects = append(result.Affects, convertAffects(a))
// 	}

// 	for _, item := range f.Evidences.Data {
// 		e, ok := item.(*resources.MqlFindingEvidence)
// 		if !ok {
// 			continue
// 		}
// 		result.Evidences = append(result.Evidences, convertFindingEvidence(e))
// 	}

// 	for _, item := range f.Remediations.Data {
// 		m, ok := item.(map[string]any)
// 		if !ok {
// 			continue
// 		}
// 		result.Remediations = append(result.Remediations, convertRemediation(m))
// 	}

// 	return result, nil
// }

// func convertFindingSource(s *resources.MqlFindingSource) *FindingSource {
// 	if s == nil {
// 		return nil
// 	}
// 	return &FindingSource{
// 		Name: s.Name.Data,
// 		Url:  s.Url.Data,
// 	}
// }

// func convertFindingDetail(d *resources.MqlFindingDetail) *FindingDetail {
// 	if d == nil {
// 		return nil
// 	}

// 	result := &FindingDetail{
// 		Category:    parseFindingCategory(d.Category.Data),
// 		Confidence:  parseConfidence(d.Confidence.Data),
// 		Description: d.Description.Data,
// 		Properties:  toStringMap(d.Properties.Data),
// 	}

// 	if d.Severity.Data != nil {
// 		result.Severity = convertFindingSeverity(d.Severity.Data)
// 	}

// 	for _, item := range d.References.Data {
// 		ref, ok := item.(*resources.MqlFindingReference)
// 		if !ok {
// 			continue
// 		}
// 		result.References = append(result.References, convertReference(ref))
// 	}

// 	return result
// }

// func convertFindingSeverity(s *resources.MqlFindingSeverity) *FindingSeverity {
// 	if s == nil {
// 		return nil
// 	}

// 	result := &FindingSeverity{
// 		Score:    float32(s.Score.Data),
// 		Severity: s.Severity.Data,
// 		Vector:   s.Vector.Data,
// 		Method:   parseScoringMethod(s.Method.Data),
// 		Rating:   parseSeverityRating(s.Rating.Data),
// 	}

// 	if s.Source.Data != nil {
// 		result.Source = convertFindingSource(s.Source.Data)
// 	}

// 	return result
// }

// func convertReference(r *resources.MqlFindingReference) *Reference {
// 	if r == nil {
// 		return nil
// 	}
// 	return &Reference{
// 		Id:       r.Id.Data,
// 		Name:     r.Name.Data,
// 		Url:      r.Url.Data,
// 		Type:     r.ReferenceType.Data,
// 		Metadata: toStringMap(r.Metadata.Data),
// 	}
// }

// func convertAffects(a *resources.MqlFindingAffects) *Affects {
// 	if a == nil {
// 		return nil
// 	}

// 	result := &Affects{}

// 	if a.Component.Data != nil {
// 		result.Component = convertComponent(a.Component.Data)
// 	}

// 	for _, item := range a.SubComponents.Data {
// 		c, ok := item.(*resources.MqlFindingComponent)
// 		if !ok {
// 			continue
// 		}
// 		result.SubComponents = append(result.SubComponents, convertComponent(c))
// 	}

// 	return result
// }

// func convertComponent(c *resources.MqlFindingComponent) *Component {
// 	if c == nil {
// 		return nil
// 	}

// 	result := &Component{
// 		Id:          c.Id.Data,
// 		Identifiers: toStringMap(c.Identifiers.Data),
// 		Properties:  toStringMap(c.Properties.Data),
// 	}

// 	if c.File.Data != nil {
// 		result.Details = &Component_File{
// 			File: convertFileComponent(c.File.Data),
// 		}
// 	}

// 	return result
// }

// func convertFileComponent(f *resources.MqlFindingFileComponent) *FileComponent {
// 	if f == nil {
// 		return nil
// 	}
// 	return &FileComponent{
// 		Path:   f.Path.Data,
// 		Hash:   f.Hash.Data,
// 		Format: f.Format.Data,
// 		Size:   f.Size.Data,
// 	}
// }

// func convertFindingEvidence(e *resources.MqlFindingEvidence) *FindingEvidence {
// 	if e == nil {
// 		return nil
// 	}

// 	result := &FindingEvidence{
// 		Confidence: parseConfidence(e.Confidence.Data),
// 		Properties: toStringMap(e.Properties.Data),
// 	}

// 	if e.Tactic.Data != nil {
// 		result.Tactic = convertAttackTactic(e.Tactic.Data)
// 	}
// 	if e.Technique.Data != nil {
// 		result.Technique = convertAttackTechnique(e.Technique.Data)
// 	}

// 	// The proto uses a oneof for evidence details. Set the first non-nil detail.
// 	switch {
// 	case e.User.Data != nil:
// 		result.Details = &FindingEvidence_User{
// 			User: convertFindingUser(e.User.Data),
// 		}
// 	case e.File.Data != nil:
// 		result.Details = &FindingEvidence_File{
// 			File: convertFindingFile(e.File.Data),
// 		}
// 	case e.Process.Data != nil:
// 		result.Details = &FindingEvidence_Process{
// 			Process: convertFindingProcess(e.Process.Data),
// 		}
// 	case e.Container.Data != nil:
// 		result.Details = &FindingEvidence_Container{
// 			Container: convertFindingContainer(e.Container.Data),
// 		}
// 	case e.Kubernetes.Data != nil:
// 		result.Details = &FindingEvidence_Kubernetes{
// 			Kubernetes: convertFindingKubernetes(e.Kubernetes.Data),
// 		}
// 	case e.RegistryKey.Data != nil:
// 		result.Details = &FindingEvidence_RegistryKey{
// 			RegistryKey: convertFindingRegistryKey(e.RegistryKey.Data),
// 		}
// 	case e.Connection.Data != nil:
// 		result.Details = &FindingEvidence_Connection{
// 			Connection: convertFindingConnection(e.Connection.Data),
// 		}
// 	}

// 	return result
// }

// func convertFindingUser(u *resources.MqlFindingUser) *FindingUser {
// 	if u == nil {
// 		return nil
// 	}
// 	return &FindingUser{
// 		Id:         u.Id.Data,
// 		Name:       u.Name.Data,
// 		Properties: toStringMap(u.Properties.Data),
// 	}
// }

// func convertFindingFile(f *resources.MqlFindingFile) *FindingFile {
// 	if f == nil {
// 		return nil
// 	}
// 	return &FindingFile{
// 		Path:     f.Path.Data,
// 		Size:     f.Size.Data,
// 		Md5:      f.Md5.Data,
// 		Sha256:   f.Sha256.Data,
// 		Contents: f.Contents.Data,
// 	}
// }

// func convertFindingProcess(p *resources.MqlFindingProcess) *FindingProcess {
// 	if p == nil {
// 		return nil
// 	}
// 	result := &FindingProcess{
// 		Cmdline: p.Cmdline.Data,
// 		Pid:     p.Pid.Data,
// 	}
// 	if p.Binary.Data != nil {
// 		result.Binary = convertFindingFile(p.Binary.Data)
// 	}
// 	if p.Script.Data != nil {
// 		result.Script = convertFindingFile(p.Script.Data)
// 	}
// 	if p.User.Data != nil {
// 		result.User = convertFindingUser(p.User.Data)
// 	}
// 	if p.Parent.Data != nil {
// 		result.Parent = convertFindingProcess(p.Parent.Data)
// 	}
// 	return result
// }

// func convertFindingContainer(c *resources.MqlFindingContainer) *FindingContainer {
// 	if c == nil {
// 		return nil
// 	}
// 	return &FindingContainer{
// 		Name:     c.Name.Data,
// 		ImageUri: c.ImageUri.Data,
// 		Digest:   c.Digest.Data,
// 	}
// }

// func convertFindingKubernetes(k *resources.MqlFindingKubernetes) *FindingKubernetes {
// 	if k == nil {
// 		return nil
// 	}

// 	result := &FindingKubernetes{}

// 	for _, item := range k.Pods.Data {
// 		pod, ok := item.(*resources.MqlFindingKubernetesPod)
// 		if !ok {
// 			continue
// 		}
// 		result.Pods = append(result.Pods, convertKubernetesPod(pod))
// 	}

// 	for _, item := range k.Nodes.Data {
// 		node, ok := item.(*resources.MqlFindingKubernetesNode)
// 		if !ok {
// 			continue
// 		}
// 		result.Nodes = append(result.Nodes, convertKubernetesNode(node))
// 	}

// 	return result
// }

// func convertKubernetesPod(p *resources.MqlFindingKubernetesPod) *FindingKubernetes_Pod {
// 	if p == nil {
// 		return nil
// 	}
// 	result := &FindingKubernetes_Pod{
// 		Name:      p.Name.Data,
// 		Namespace: p.Namespace.Data,
// 	}
// 	for _, item := range p.Containers.Data {
// 		c, ok := item.(*resources.MqlFindingContainer)
// 		if !ok {
// 			continue
// 		}
// 		result.Containers = append(result.Containers, convertFindingContainer(c))
// 	}
// 	return result
// }

// func convertKubernetesNode(n *resources.MqlFindingKubernetesNode) *FindingKubernetes_Node {
// 	if n == nil {
// 		return nil
// 	}
// 	return &FindingKubernetes_Node{
// 		Name: n.Name.Data,
// 		Id:   n.Id.Data,
// 	}
// }

// func convertFindingRegistryKey(r *resources.MqlFindingRegistryKey) *FindingRegistryKey {
// 	if r == nil {
// 		return nil
// 	}
// 	return &FindingRegistryKey{
// 		Path: r.Path.Data,
// 		Name: r.Name.Data,
// 		Data: r.Data.Data,
// 	}
// }

// func convertFindingConnection(c *resources.MqlFindingConnection) *FindingConnection {
// 	if c == nil {
// 		return nil
// 	}
// 	return &FindingConnection{
// 		DestinationAddress: c.DestinationAddress.Data,
// 		DestinationPort:    int32(c.DestinationPort.Data),
// 		SourceAddress:      c.SourceAddress.Data,
// 		SourcePort:         int32(c.SourcePort.Data),
// 		Protocol:           parseConnectionProtocol(c.Protocol.Data),
// 	}
// }

// func convertAttackTactic(t *resources.MqlFindingAttackTactic) *AttackTactic {
// 	if t == nil {
// 		return nil
// 	}
// 	return &AttackTactic{
// 		Id:          t.Id.Data,
// 		Name:        t.Name.Data,
// 		Description: t.Description.Data,
// 	}
// }

// func convertAttackTechnique(t *resources.MqlFindingAttackTechnique) *AttackTechnique {
// 	if t == nil {
// 		return nil
// 	}
// 	return &AttackTechnique{
// 		Id:          t.Id.Data,
// 		Name:        t.Name.Data,
// 		Description: t.Description.Data,
// 	}
// }

// func convertRemediation(m map[string]any) *FindingRemediation {
// 	r := &FindingRemediation{}
// 	if v, ok := m["category"].(string); ok {
// 		r.Category = parseRemediationCategory(v)
// 	}
// 	if v, ok := m["summary"].(string); ok {
// 		r.Summary = v
// 	}
// 	if v, ok := m["details"].(string); ok {
// 		r.Details = v
// 	}
// 	if v, ok := m["fix_type"].(string); ok {
// 		r.FixType = v
// 	}
// 	if v, ok := m["fixType"].(string); ok && r.FixType == "" {
// 		r.FixType = v
// 	}
// 	if v, ok := m["fix_id"].(string); ok {
// 		r.FixId = v
// 	}
// 	if v, ok := m["fixId"].(string); ok && r.FixId == "" {
// 		r.FixId = v
// 	}
// 	if v, ok := m["url"].(string); ok {
// 		r.Url = v
// 	}
// 	return r
// }

// // Enum parsing helpers

// func parseFindingStatus(s string) FindingStatus {
// 	switch strings.ToLower(s) {
// 	case "not_affected", "not affected", "notaffected":
// 		return FindingStatus_FINDING_STATUS_NOT_AFFECTED
// 	case "affected":
// 		return FindingStatus_FINDING_STATUS_AFFECTED
// 	case "fixed":
// 		return FindingStatus_FINDING_STATUS_FIXED
// 	case "under_investigation", "under investigation", "underinvestigation":
// 		return FindingStatus_FINDING_STATUS_UNDER_INVESTIGATION
// 	case "false_positive", "false positive", "falsepositive":
// 		return FindingStatus_FINDING_STATUS_FALSE_POSITIVE
// 	case "wont_fix", "wont fix", "wontfix":
// 		return FindingStatus_FINDING_STATUS_WONT_FIX
// 	default:
// 		return FindingStatus_FINDING_STATUS_UNSPECIFIED
// 	}
// }

// func parseFindingCategory(s string) FindingDetail_Category {
// 	switch strings.ToLower(s) {
// 	case "security":
// 		return FindingDetail_CATEGORY_SECURITY
// 	case "vulnerability":
// 		return FindingDetail_CATEGORY_VULNERABILITY
// 	case "advisory":
// 		return FindingDetail_CATEGORY_ADVISORY
// 	case "threat":
// 		return FindingDetail_CATEGORY_THREAT
// 	case "malware":
// 		return FindingDetail_CATEGORY_MALWARE
// 	case "informational":
// 		return FindingDetail_CATEGORY_INFORMATIONAL
// 	default:
// 		return FindingDetail_CATEGORY_UNSPECIFIED
// 	}
// }

// func parseConfidence(s string) Confidence {
// 	switch strings.ToLower(s) {
// 	case "low":
// 		return Confidence_CONFIDENCE_LOW
// 	case "medium":
// 		return Confidence_CONFIDENCE_MEDIUM
// 	case "high":
// 		return Confidence_CONFIDENCE_HIGH
// 	default:
// 		return Confidence_CONFIDENCE_UNSPECIFIED
// 	}
// }

// func parseScoringMethod(s string) ScoringMethod {
// 	switch strings.ToLower(s) {
// 	case "cvssv2", "cvss_v2":
// 		return ScoringMethod_SCOREMETHOD_CVSSv2
// 	case "cvssv3", "cvss_v3":
// 		return ScoringMethod_SCOREMETHOD_CVSSv3
// 	case "cvssv4", "cvss_v4":
// 		return ScoringMethod_SCOREMETHOD_CVSSv4
// 	case "epss":
// 		return ScoringMethod_SCOREMETHOD_EPSS
// 	case "ssvc":
// 		return ScoringMethod_SCOREMETHOD_SSVC
// 	case "ubuntu":
// 		return ScoringMethod_SCOREMETHOD_UBUNTU
// 	default:
// 		return ScoringMethod_SCOREMETHOD_UNSPECIFIED
// 	}
// }

// func parseSeverityRating(s string) SeverityRating {
// 	switch strings.ToLower(s) {
// 	case "critical":
// 		return SeverityRating_SEVERITY_RATING_CRITICAL
// 	case "high":
// 		return SeverityRating_SEVERITY_RATING_HIGH
// 	case "medium":
// 		return SeverityRating_SEVERITY_RATING_MEDIUM
// 	case "low":
// 		return SeverityRating_SEVERITY_RATING_LOW
// 	case "none":
// 		return SeverityRating_SEVERITY_RATING_NONE
// 	default:
// 		return SeverityRating_SEVERITY_RATING_UNSPECIFIED
// 	}
// }

// func parseConnectionProtocol(s string) FindingConnection_ConnectionProtocol {
// 	switch strings.ToLower(s) {
// 	case "icmp":
// 		return FindingConnection_ICMP
// 	case "tcp":
// 		return FindingConnection_TCP
// 	case "udp":
// 		return FindingConnection_UDP
// 	default:
// 		return FindingConnection_UNSPECIFIED
// 	}
// }

// func parseRemediationCategory(s string) FindingRemediation_Category {
// 	switch strings.ToLower(s) {
// 	case "no_fix_planned", "nofixplanned":
// 		return FindingRemediation_NoFixPlanned
// 	case "none_available", "noneavailable":
// 		return FindingRemediation_NoneAvailable
// 	case "fix":
// 		return FindingRemediation_Fix
// 	case "workaround":
// 		return FindingRemediation_Workaround
// 	default:
// 		return FindingRemediation_Unspecified
// 	}
// }

// // toStringMap converts a map[string]any to map[string]string.
// func toStringMap(m map[string]any) map[string]string {
// 	if m == nil {
// 		return nil
// 	}
// 	result := make(map[string]string, len(m))
// 	for k, v := range m {
// 		result[k] = fmt.Sprintf("%v", v)
// 	}
// 	return result
// }
