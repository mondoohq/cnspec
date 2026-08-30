# HBN Network Content Plan

## Scope

Add cnspec policy, querypack, documentation, and validation coverage for HBN network inventory, secondary-interface policy coverage, egress routing, NAT visibility, and internet exposure. MQL resources and operator RBAC/deployment are covered in separate PRs.

## Implementation status

This PR adds the first cnspec content slice for the normalized Kubernetes network posture schema:

- Inventory queries in `content/querypacks/mondoo-kubernetes-inventory.mql.yaml` for `k8s.networkExposures`, normalized Gateway API gateway/route exposure evidence, `k8s.egressRoutes`, `k8s.egressNats`, and `k8s.networkPolicyCoverages`.
- A dedicated preview policy in `content/mondoo-kubernetes-network-posture.mql.yaml` covering internet exposure evidence, public egress classification, public egress NAT ownership, secondary-interface policy coverage, primary-interface egress default-deny posture, and AdminNetworkPolicy/BaselineAdminNetworkPolicy deny semantics.

The content targets normalized resources from the HBN/network MQL provider PR. That provider derives internet exposure, Gateway API route, egress route, NAT, and policy coverage summaries from Kubernetes Services and Ingresses, Gateway API resources, AdminNetworkPolicy/BaselineAdminNetworkPolicy, HBN CRDs, Coil `Egress`, MultiNetworkPolicy, Calico, and Cilium objects. Raw drill-down resources such as `k8s.hbn.vrf`, `k8s.hbn.network`, and raw `k8s.multiNetworkPolicy` remain follow-up content if the provider exposes them directly.

## Goals

- Show HBN intent and compiled status alongside ordinary Kubernetes network resources.
- Make internet-exposed services discoverable from Kubernetes Service, Ingress, Gateway API, and HBN signals.
- Make egress NAT, public destinations, and route classifications visible.
- Show native NetworkPolicy, AdminNetworkPolicy/BaselineAdminNetworkPolicy, MultiNetworkPolicy, Calico, and Cilium coverage in one policy view.
- Include optional Calico Whisker and Cilium flow evidence when configured.
- Keep content safe on clusters without HBN, MultiNetworkPolicy, Calico, or Cilium.

## Non-goals

- Do not enforce or mutate network policy.
- Do not require observed flow integrations.
- Do not claim packet-level enforcement from static API state alone.
- Do not fail ordinary Kubernetes users because HBN resources are missing.

## Expected MQL inputs

The first content slice uses the normalized MQL resources listed below and must guard them for backwards compatibility:

- `k8s.networkExposures`
- `k8s.egressRoutes`
- `k8s.egressNats`
- `k8s.networkPolicyCoverages`

Later drill-down content may add raw resource queries for HBN intent/status, NetworkAttachmentDefinitions, MultiNetworkPolicy manifests, BGP peerings, traffic mirrors, and collectors once the provider exposes dedicated raw resources.

## Inventory querypack

Extend `content/querypacks/mondoo-kubernetes-inventory.mql.yaml` or add a dedicated network inventory querypack.

Inventory sections for this slice:

- Internet exposure summary.
- Normalized Gateway API gateway and route exposure inventory.
- Egress NAT summary.
- Egress route summary.
- Network policy coverage summary, including native, admin, secondary-interface, Calico, and Cilium policy references.

The querypack prioritizes summarized posture resources for dashboards. Raw manifests should be added later only as drill-down evidence to avoid duplicate findings for the same network issue.

## Policy content

Add a dedicated policy after MQL resource names are final, for example:

- `content/mondoo-kubernetes-hbn-network-security.mql.yaml`

Candidate controls:

1. Internet-exposed services are explicitly classified.
   - Fail when `k8s.networkExposure.internetExposed == true` and no approved classification or owner metadata exists.
   - Evidence should include source kind, namespace/name, addresses, ports, exposure reason, and confidence.

2. Internet exposure must come from declared HBN inbound intent when HBN is enabled.
   - Fail when a public exposure exists without matching `k8s.hbn.inbound` or approved exception metadata.
   - Skip when HBN resources are absent.

3. Public egress routes are classified and owned.
   - Fail when `k8s.egressRoute.publicCidrs` is non-empty and classification/owner is missing.

4. Egress NAT is visible and approved.
   - Fail when `k8s.egressNat` uses public addresses without approved route classification.
   - Warn when NAT is inferred with low confidence.

5. Secondary pod interfaces have MultiNetworkPolicy coverage.
   - Fail when a pod has secondary interfaces and `secondaryInterfaceIngressCovered` or `secondaryInterfaceEgressCovered` is false.
   - Evidence should include pod, network attachment, and missing direction.

6. Native NetworkPolicy coverage exists for primary interfaces.
   - Reuse existing Kubernetes security policy patterns where possible.
   - Link to `k8s.networkPolicyCoverage`.

7. AdminNetworkPolicy/BaselineAdminNetworkPolicy deny semantics are explicit.
   - Fail when admin policy references exist but no catch-all ingress or egress deny guardrail is present.
   - Do not treat scoped `Deny`, `Allow`, or `Pass` rules as default-deny posture.

8. Traffic mirrors and collectors are explicitly approved.
   - Fail when `k8s.hbn.trafficMirror` or `k8s.hbn.collector` exists without owner, purpose, and approved destination metadata.

9. BGP peerings that announce public CIDRs are reviewed.
   - Fail when public announcements exist without approved classification.
   - Include peer ASN, node/network reference, and CIDRs when available.

10. Optional observed flows match declared policy.
   - Informational or warning at first.
   - Use Calico Whisker or Cilium Hubble evidence only when configured.

## Exception and waiver behavior

Use normal Mondoo policy exceptions for accepted public exposure, egress NAT, traffic mirrors, and missing coverage during migration. Controls should include stable asset/query identity so exceptions remain attached to the network object, not a transient pod instance, whenever possible.

Stable identities should prefer:

- HBN intent object UID for declared routes and exposure.
- Service, Ingress, or Gateway UID for Kubernetes exposure.
- NetworkAttachmentDefinition UID plus workload owner for secondary-interface coverage.
- Digest-like normalized route id for generated egress summaries only when no source object exists.

## Report UX

Group findings by network concern:

- Internet exposure.
- Egress and NAT.
- Policy coverage.
- Observability and mirrors.
- HBN inventory health.

Each finding should show:

- Affected namespace/name.
- Source object kind.
- Network or VRF.
- Direction: ingress or egress.
- Public/private classification.
- Policy coverage status.
- Confidence and reason.

Avoid producing both raw-resource and normalized-resource failures for the same issue. Raw HBN resources should provide drill-down; normalized posture controls should be the main failing controls.

## Documentation

Add docs after implementation:

- What HBN network inventory collects.
- How internet exposure is classified.
- How egress NAT is represented.
- How MultiNetworkPolicy differs from native NetworkPolicy.
- How to enable optional Whisker and Hubble flow evidence.
- Known limitations:
  - Static API state cannot prove dataplane enforcement.
  - Observed flows may be sampled or time-bounded.
  - Secondary-interface policy semantics depend on the installed controller.

## Implementation phases

### Phase 1: Inventory querypack

- Add guarded inventory queries for HBN and MultiNetworkPolicy.
- Add summarized exposure, egress, and policy coverage queries.
- Add fixtures for HBN present, HBN absent, MultiNetworkPolicy present, and secondary interfaces present.

### Phase 2: Preview policy

- Add low-noise controls for internet exposure, public egress, and secondary-interface policy coverage.
- Keep controls skipped when resources are unavailable.
- Add remediation text that points users to HBN intent resources and MultiNetworkPolicy.

### Phase 3: Expanded network controls

- Add NAT, BGP public announcements, traffic mirrors, and collectors.
- Add stable exception identities.
- Add warning/informational controls for observed flows.

### Phase 4: Documentation and examples

- Add docs and example screenshots or sample output after the resources exist.
- Add example policies showing how teams can require labels such as `mondoo.com/owner`, `network.t-caas.telekom.com/classification`, or their own taxonomy.

## Test plan

Focused tests:

- `make test/lint/content`
- `go test ./content/...`.
- `go test ./policy/...` when bundle or policy metadata changes.
- Hard bundle compile tests for every new querypack and policy query. Missing provider resources or fields must fail tests instead of being skipped.
- Fixture tests for:
  - public LoadBalancer without classification
  - Gateway API route with public hostname or address
  - AdminNetworkPolicy/BaselineAdminNetworkPolicy with and without catch-all deny semantics
  - public HBN Inbound with classification
  - secondary interface without MultiNetworkPolicy
  - secondary interface with ingress and egress MultiNetworkPolicy
  - public egress NAT without owner
  - traffic mirror without approval metadata
  - cluster without HBN CRDs

Full verification:

- `make test`

Manual verification after MQL/operator PRs exist:

- Install sample HBN CRDs from the intent-based network operator PR.
- Install sample MultiNetworkPolicy resources from `telekom/multi-networkpolicy-nftables`.
- Install Gateway API CRDs and sample HTTPRoute/TCPRoute/TLSRoute/UDPRoute resources.
- Install AdminNetworkPolicy/BaselineAdminNetworkPolicy sample resources.
- Run `cnspec scan k8s --discover clusters` through the operator or CLI. The network posture policy is cluster-scoped; workload, pod, and controller child assets do not evaluate this policy directly.
- Confirm internet exposure, egress NAT, and secondary-interface policy coverage appear in Mondoo.

## Acceptance criteria

- Inventory shows normalized Gateway API routing, supported HBN intent, AdminNetworkPolicy/BaselineAdminNetworkPolicy, MultiNetworkPolicy, Calico, Cilium, secondary-interface coverage, exposure, egress routes, and NAT summaries. Raw HBN node status, traffic mirror, collector, and BGP drill-down resources remain follow-up content.
- Policies flag unclassified internet exposure, missing secondary-interface policy coverage, missing primary egress isolation, and admin policy records without catch-all deny semantics.
- Policies use stable identities so normal Mondoo exceptions work for accepted exposure and migration gaps.
- Content produces empty results cleanly on clusters without optional CRDs while compile tests still fail on provider schema drift.
- Content lint and bundle tests cover new querypacks and policies.
