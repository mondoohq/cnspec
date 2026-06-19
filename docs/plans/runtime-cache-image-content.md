# Runtime Cache Image Content Plan

## Scope

Add cnspec policy, querypack, docs, and validation coverage for node-local runtime-cache image scanning. MQL provider resources and operator deployment are covered in separate PRs.

## Implementation status

This PR adds the first cnspec content slice:

- Inventory queries in `content/querypacks/mondoo-kubernetes-inventory.mql.yaml` for runtime delegates, runtime cached images, and pod-to-runtime-image coverage.
- A dedicated preview policy in `content/mondoo-kubernetes-runtime-image-cache.mql.yaml` for runtime delegate readiness, read-only/no-pull delegate posture, pod image match status, and immutable image identity.

The content depends on the runtime-cache MQL schema from the mql provider PR. Until cnspec bumps to a module version containing `container.runtimeDelegate`, `container.runtimeImage`, `container.runtimeImageLayer`, `k8s.node.runtimeDelegates`, `k8s.node.runtimeImages`, `k8s.pod.containerStatuses[].runtimeImage`, and `k8s.pod.containerStatuses[].runtimeImageStatus`, linting this content requires a temporary local module replacement to that MQL branch. The schema fixture in this PR mirrors that provider contract so bundle tests can validate the draft content before the dependency is released; remove the fixture-only coupling once cnspec consumes the released MQL module.

## Goals

- Make node-local cached image scan coverage visible to users.
- Add inventory content that shows which images are in use, which were scanned, and why any were skipped.
- Add policies that detect runtime-cache scanner gaps without requiring registry credentials.
- Keep existing registry and container image policies working unchanged.
- Provide fixtures and content tests that exercise protected-registry and no-pull semantics.

## Non-goals

- Do not implement runtime access in cnspec content.
- Do not require the runtime-cache scanner for all Kubernetes users.
- Do not duplicate existing vulnerability policies when the scanner already reports package and vuln data for the image asset.

## Expected MQL inputs

This plan assumes the MQL PR adds resources equivalent to:

- `container.runtimeDelegate`
- `container.runtimeImage`
- `container.runtimeImageLayer`
- `k8s.node.runtimeDelegates`
- `k8s.node.runtimeImages`
- `k8s.pod.containers[].runtimeImage`
- `k8s.pod.containers[].runtimeImageStatus`

Runtime-cache content also assumes the paired provider resolves configured runtime delegates from the scan connection and matches shared runtime-image resources by immutable identity. That keeps scan results deduplicated by digest instead of workload tag or pod instance.

Policy content should be written defensively so missing resources do not fail older providers.

## Inventory querypack

Add or extend a Kubernetes inventory querypack, likely `content/querypacks/mondoo-kubernetes-inventory.mql.yaml`.

New sections:

- Runtime image cache delegates by node.
- Runtime images by node.
- Images in use by workloads.
- Image scan coverage.
- Images skipped by reason.

Example query concepts:

```mql
k8s.nodes {
  name
  runtimeDelegates {
    id
    kind
    status
    statusMessage
  }
  runtimeImages {
    resolvedDigest
    repoTags
    repoDigests
    inUse
    scanStatus
    scanStatusMessage
  }
}
```

```mql
k8s.pods {
  namespace
  name
  containers {
    name
    image
    imageID
    runtimeImageStatus
    runtimeImage {
      resolvedDigest
      scanStatus
    }
  }
}
```

## Policy content

Add a dedicated policy file only after MQL resource names are final, for example:

- `content/mondoo-kubernetes-runtime-image-cache.mql.yaml`

Candidate controls:

1. Runtime-cache scanning is enabled on every schedulable node.
   - Fail when a schedulable node has no ready `container.runtimeDelegate`.
   - Skip control when runtime-cache resources are unavailable.

2. In-use pod images are matched to a node-local runtime image.
   - Fail when `runtimeImageStatus` is `notPresent`, `runtimeUnavailable`, or `ambiguous`.
   - Include workload namespace/name/container in evidence.

3. In-use pod images are scanned from immutable local image identity.
   - Fail when the runtime image has no `resolvedDigest` or equivalent immutable ID.

4. Runtime-cache scanner does not rely on registry credentials.
   - Pass when scan evidence comes from `container-runtime-image` or equivalent no-pull source.
   - Fail only when runtime-cache mode is enabled and image evidence shows a registry pull path.

5. Stale cached images are visible.
   - Informational control listing cached images with `inUse=false`.
   - This should not fail by default because runtime garbage collection is cluster-specific.

6. Runtime delegate health is clean.
   - Fail when delegates report `permissionDenied`.
   - Warn when delegates are `unavailable` and another delegate covered the node.

## Result UX

Policies should group evidence by:

- Node.
- Runtime delegate.
- Workload.
- Image digest.

Avoid presenting duplicate failures for every pod using the same image. Prefer one image-level failure with workload references when MQL supports it.

## Documentation

Update supply-chain or Kubernetes scanning docs after implementation:

- Explain the difference between registry scanning and node-local runtime-cache scanning.
- State clearly that runtime-cache scanning does not pull images.
- State clearly that runtime-cache scanning does not read image pull secrets.
- Document limitations:
  - It only scans images present on nodes.
  - It only sees nodes where the DaemonSet can access the runtime.
  - Runtime socket access is privileged host access even when the scanner is read-only.

## Implementation phases

### Phase 1: Inventory content

- Add querypack entries guarded by resource availability.
- Add fixtures with:
  - ready delegate and scanned image
  - missing delegate
  - protected-registry image scanned from cache
  - image referenced by pod but not present on node

### Phase 2: Policy controls

- Add the dedicated runtime-cache policy.
- Keep severity modest for preview controls until field data confirms low false positives.
- Use clear remediation text that points users to the operator runtime-cache scanner config.

### Phase 3: Documentation

- Update `docs/supplychain/docker.mdx` or the Kubernetes scan docs with runtime-cache mode.
- Add examples for enabling runtime-cache scanning through the operator.

### Phase 4: Bundle and lint integration

- Ensure new policy and querypack are included in content bundle tests.
- Add testdata for pass and fail outcomes.

## Test plan

Focused tests:

- `make test/lint/content`
- `go test ./content/...` if content package tests are available.
- `go test ./policy/...` for bundle loading and policy metadata if touched.
- Fixture tests for pass/fail runtime-cache controls.
- Golden tests for policy evidence text when the content framework supports it.

Full verification:

- `make test`

Manual verification after MQL/operator PRs exist:

- Run a k3d cluster with a locally loaded image that is not pullable from the registry.
- Enable runtime-cache scanner.
- Confirm cnspec reports the image as scanned and does not require an image pull secret.
- Confirm a pod image absent from the runtime cache appears as `notPresent`.

## Acceptance criteria

- Users can see runtime delegates, cached images, scan status, and skipped reasons in inventory.
- Policies distinguish missing local images from registry authentication failures.
- Content does not fail on older providers where runtime-cache resources do not exist.
- Content lint and bundle tests cover new controls and querypack entries.
