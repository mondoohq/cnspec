# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# REST API curl validation against vendor OpenAPI specs.
#
# Several SaaS products our policies cover have no usable CLI for the
# settings the checks target — Cloudflare's `flarectl` is read-only,
# Tailscale/Slack/Atlassian/Grafana admin surfaces are API-first — so the
# realistic remediation path is `curl` against the vendor's REST API. For
# every vendor that publishes an OpenAPI (or Swagger 2.0) spec, this
# module scans the policy's bash blocks for curl calls against that
# vendor's API host and verifies each call's URL path + HTTP method —
# and, where the call sends a JSON payload, the request body — against
# the spec.
#
# Body validation covers the subset of JSON Schema these specs actually
# use for the endpoints in our policies: $ref, allOf, oneOf/anyOf, type,
# enum, properties/required, and array items. Angle-bracket placeholders
# (`"<account-name>"`) and environment-variable placeholders (`"$ORG_ID"`)
# act as wildcards that satisfy any schema.
#
# Spec sources come in two flavors, mirroring how the CLI validators
# source command data:
#
#   - "url" + "pin": downloaded at validation time from a URL pinned to
#     an immutable ref (a git commit SHA baked into a raw GitHub URL) and
#     cached under ~/.cache. Bumping the pin is a deliberate maintainer
#     action, like bumping doctl or the AWS CLI version.
#   - "file": checked into cmd_data/ because the vendor serves the spec
#     from a live, unversioned endpoint that can change (or break)
#     without notice. Regenerate with dump_api_specs.py; never hand-edit.

import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path
from urllib.parse import urlparse

from .common import (
    CMD_DATA_DIR,
    FAILURES,
    SCRIPT_DIR,
    extract_bash_blocks,
    policy_relpath,
    split_commands,
)

# ---------------------------------------------------------------------------
# Spec loading
# ---------------------------------------------------------------------------

# None of these upstream repos ship releases — `main`/`master` is the only
# branch and it moves — so each raw URL pins a known-good commit SHA.
CLOUDFLARE_OPENAPI_SHA = "a0e7cfa11b0d08b05a0b373f47ea722bd48ca7c4"
SLACK_OPENAPI_SHA = "bc08db49625630e3585bf2f1322128ea04f2a7f3"
GRAFANA_OPENAPI_SHA = "8c7e01c44c7afd14f7143589840bbd820a4195f9"


def _spec_cache_path(name: str, pin: str) -> Path:
    cache_dir = Path.home() / ".cache" / "cnspec-validation"
    cache_dir.mkdir(parents=True, exist_ok=True)
    return cache_dir / f"{name}-openapi-{pin}.json"


def _load_spec(source: dict) -> dict:
    """Return a parsed spec document for one provider spec source.

    source is either {"name", "file"} for a checked-in cmd_data spec or
    {"name", "url", "pin"} for a pinned remote spec (downloaded once and
    cached under ~/.cache).
    """
    if "file" in source:
        path = CMD_DATA_DIR / source["file"]
        if not path.exists():
            print(
                f"Error: checked-in API spec not found: {path}\n"
                "Regenerate it with:\n"
                f"  python3 {SCRIPT_DIR / 'dump_api_specs.py'}",
                file=sys.stderr,
            )
            sys.exit(1)
        return json.loads(path.read_text())

    cache = _spec_cache_path(source["name"], source["pin"][:12])
    if not cache.exists():
        print(
            f"Fetching {source['name']} OpenAPI spec (pin {source['pin'][:12]})...",
            file=sys.stderr,
        )
        try:
            with urllib.request.urlopen(source["url"], timeout=60) as r:
                cache.write_bytes(r.read())
        except (urllib.error.URLError, TimeoutError) as e:
            print(
                f"Error: failed to download {source['name']} OpenAPI spec from\n"
                f"  {source['url']}\n"
                f"  ({e})\n"
                "\n"
                "If you are behind a proxy or air-gapped, manually download the\n"
                f"spec and save it to:\n  {cache}",
                file=sys.stderr,
            )
            sys.exit(1)
    try:
        return json.loads(cache.read_text())
    except json.JSONDecodeError as e:
        print(
            f"Error: cached {source['name']} OpenAPI spec is corrupted\n"
            f"  ({cache}: {e})\n"
            "\n"
            "Delete the cache file and re-run to re-download:\n"
            f"  rm {cache}",
            file=sys.stderr,
        )
        sys.exit(1)


def _spec_mount(spec: dict) -> str:
    """Base path the spec's path templates are mounted under.

    OpenAPI 3 declares it as the path component of servers[0].url (which
    may be a relative URL like "/api"); Swagger 2.0 uses basePath. A curl
    URL's path must start with the mount; the remainder is matched
    against the spec's path templates.
    """
    if "basePath" in spec:  # Swagger 2.0
        return spec["basePath"].rstrip("/")
    servers = spec.get("servers") or []
    if servers and servers[0].get("url"):
        return urlparse(servers[0]["url"]).path.rstrip("/")
    return ""


def _index_spec_paths(spec: dict) -> dict[str, set[str]]:
    """Index a spec's paths as {path_template: {HTTP_METHOD, ...}}."""
    out: dict[str, set[str]] = {}
    for path, ops in spec.get("paths", {}).items():
        methods = {m.upper() for m in ops if m in (
            "get", "post", "put", "patch", "delete", "head", "options",
        )}
        if methods:
            out[path] = methods
    return out


# A leading `/vN` API version segment. Vercel versions every endpoint in the
# URL (`/v9/projects/{id}`, `/v3/user/tokens/{id}`) and keeps several
# versions live simultaneously, but its OpenAPI spec documents only one
# version per operation — often a different one than the policy's remediation
# uses (the token endpoints are `/v6` and `/v3` in the spec, `/v5` in the
# docs). Matching the version literally would reject valid calls, so for
# providers that set `strip_api_version` we normalize the version away on
# both sides. See _merge_version_stripped_paths.
_API_VERSION_RE = re.compile(r"^/v\d+(?=/)")


def _strip_api_version(path: str) -> str:
    return _API_VERSION_RE.sub("", path)


def _merge_version_stripped_paths(spec: dict) -> None:
    """Add version-stripped aliases of every `/vN/...` path into the spec's
    `paths`, in place, so a curl call whose version differs from the spec's
    still resolves — and its request-body schema (looked up by path
    template) is still found.

    When two versions of the same base path each document a different
    method (`GET /v6/user/tokens`, `POST /v3/user/tokens`), their methods
    coexist on the merged `/user/tokens`. When both document the same
    method, the higher version wins, so we merge in ascending version
    order.
    """
    paths = spec.setdefault("paths", {})

    def version(p: str) -> int:
        m = re.match(r"^/v(\d+)/", p)
        return int(m.group(1)) if m else -1

    versioned = sorted(
        (p for p in list(paths) if _API_VERSION_RE.match(p)), key=version
    )
    for p in versioned:
        stripped = _strip_api_version(p)
        if stripped == p:
            continue
        dst = paths.setdefault(stripped, {})
        for method, op in paths[p].items():
            dst[method] = op  # ascending order => higher version wins


# ---------------------------------------------------------------------------
# Path matching
# ---------------------------------------------------------------------------

# A URL path segment that stands in for a concrete value in remediation
# examples: <zone-id>, $ORG_ID, ${ORG_ID}, or a literal {param} copied
# from vendor docs. These match any OpenAPI {param} template segment.
_PLACEHOLDER_SEGMENT_RE = re.compile(
    r"^(<[A-Za-z0-9_.-]+>|\$\{?[A-Za-z_][A-Za-z0-9_]*\}?|\{[A-Za-z0-9_.-]+\})$"
)


def _segment_matches(tmpl_seg: str, curl_seg: str) -> bool:
    if tmpl_seg == curl_seg:
        return True
    if tmpl_seg.startswith("{") and tmpl_seg.endswith("}"):
        # Templated segment matches any concrete value or placeholder in
        # the curl URL.
        return curl_seg != "" and "/" not in curl_seg
    return False


def _template_specificity(tmpl: str) -> int:
    """Number of literal (non-{param}) segments in a path template."""
    return sum(
        1 for p in tmpl.split("/")
        if p and not (p.startswith("{") and p.endswith("}"))
    )


def _match_spec_path(curl_path: str, paths_index: dict[str, set[str]]) -> str | None:
    """Find a path template matching a concrete curl URL path.

    When multiple templates match, prefers the one with the most literal
    (non-`{param}`) segments — so e.g. `/zones/{id}/dns_records/import`
    wins over `/zones/{id}/dns_records/{record_id}` for a curl call to
    `/zones/<zone-id>/dns_records/import`. Returns the matched path
    template, or None.
    """
    if curl_path in paths_index:
        return curl_path

    curl_parts = curl_path.split("/")
    best: str | None = None
    best_specificity = -1
    for tmpl in paths_index:
        tmpl_parts = tmpl.split("/")
        if len(tmpl_parts) != len(curl_parts):
            continue
        if not all(_segment_matches(t, c) for t, c in zip(tmpl_parts, curl_parts)):
            continue
        specificity = _template_specificity(tmpl)
        if specificity > best_specificity:
            best = tmpl
            best_specificity = specificity
    return best


# ---------------------------------------------------------------------------
# curl parsing
# ---------------------------------------------------------------------------

# -X / --request may appear before or after the URL.
_CURL_METHOD_RE = re.compile(r"(?:^|\s)(?:-X|--request)(?:\s+|=)([A-Z]+)")

# Any curl flag that supplies a request body. split_commands() has already
# shlex-tokenized and re-joined the command, so the payload appears after
# the flag with its shell quoting stripped but its JSON quoting intact.
_CURL_DATA_RE = re.compile(
    r"(?:^|\s)(?:--data(?:-raw|-binary|-ascii)?|--json|-d)(?:\s+|=)"
)

# A whole-string placeholder value like "<account-name>" or "$ORG_ID".
# During body validation these act as wildcards that satisfy any schema.
_PLACEHOLDER_VALUE_RE = re.compile(
    r"^(<[A-Za-z0-9_.-]+>|\$\{?[A-Za-z_][A-Za-z0-9_]*\}?)$"
)

# Quote a bare (unquoted) placeholder so the payload parses as JSON, e.g.
# {"max_age": <seconds>} -> {"max_age": "<seconds>"}. Placeholders already
# inside JSON strings are excluded by the quote look-around.
_BARE_PLACEHOLDER_RE = re.compile(r'(?<![\w"<])<([A-Za-z0-9_.-]+)>(?![\w">])')


def parse_api_curl(cmd: str, host: str) -> tuple[str, str, str | None] | None:
    """Extract (HTTP_METHOD, URL_PATH, BODY) from a curl call against host.

    Returns None if the command is not a curl call against the given API
    host. The method defaults to GET when -X / --request is absent
    (matches curl's default when no body is supplied; for POST/PATCH/PUT
    we always require an explicit -X in remediation snippets). BODY is
    the raw --data payload, or None when the command sends no body.
    """
    if host not in cmd:
        return None
    if not cmd.lstrip().startswith("curl"):
        return None

    url_match = re.search(re.escape(host) + r"(/[^\s'\"]+)", cmd)
    if not url_match:
        return None
    path = url_match.group(1)
    # Drop a fragment or query string if present, and a trailing slash —
    # spec path templates never carry one, and the segment-wise matcher
    # would otherwise see a spurious empty segment.
    path = path.split("?", 1)[0].split("#", 1)[0]
    if len(path) > 1:
        path = path.rstrip("/")

    method_match = _CURL_METHOD_RE.search(cmd)
    method = method_match.group(1).upper() if method_match else "GET"

    return method, path, _extract_curl_body(cmd)


def _extract_curl_body(cmd: str) -> str | None:
    """Return the payload passed via --data/-d/--json, or None.

    The command string has been shlex-tokenized and space-joined, so the
    payload's shell quotes are gone. A JSON object/array payload may
    therefore contain spaces; scan to its balanced closing brace instead
    of splitting on whitespace.
    """
    m = _CURL_DATA_RE.search(cmd)
    if not m:
        return None
    rest = cmd[m.end():]
    if not rest:
        return None
    if rest[0] in "{[":
        close = {"{": "}", "[": "]"}[rest[0]]
        depth = 0
        in_str = False
        escaped = False
        for i, ch in enumerate(rest):
            if escaped:
                escaped = False
            elif ch == "\\":
                escaped = True
            elif ch == '"':
                in_str = not in_str
            elif not in_str:
                if ch == rest[0]:
                    depth += 1
                elif ch == close:
                    depth -= 1
                    if depth == 0:
                        return rest[: i + 1]
        return rest  # unbalanced; json.loads will report the details
    return rest.split(None, 1)[0]


# ---------------------------------------------------------------------------
# JSON-Schema-subset body validation
# ---------------------------------------------------------------------------

def _resolve_ref(node, spec: dict):
    """Follow a chain of $ref JSON pointers within the spec document."""
    seen: set[str] = set()
    while isinstance(node, dict) and "$ref" in node:
        ref = node["$ref"]
        if ref in seen:
            return {}
        seen.add(ref)
        cur = spec
        for part in ref.lstrip("#/").split("/"):
            part = part.replace("~1", "/").replace("~0", "~")
            cur = cur.get(part, {}) if isinstance(cur, dict) else {}
        node = cur
    return node


def _type_ok(value, t) -> bool:
    if isinstance(t, list):
        return any(_type_ok(value, x) for x in t)
    if t == "object":
        return isinstance(value, dict)
    if t == "array":
        return isinstance(value, list)
    if t == "string":
        return isinstance(value, str)
    if t == "boolean":
        return isinstance(value, bool)
    if t == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if t == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if t == "null":
        return value is None
    return True


def _json_type_name(value) -> str:
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "boolean"
    if isinstance(value, int):
        return "integer"
    if isinstance(value, float):
        return "number"
    if isinstance(value, str):
        return "string"
    if isinstance(value, list):
        return "array"
    return "object"


def _validate_body(
    value,
    schema,
    spec: dict,
    loc: str,
    errors: list[str],
    check_required: bool = True,
) -> None:
    """Validate a parsed JSON value against a (subset of) JSON Schema.

    Covers what the vendor specs use for the request bodies in our
    policies: $ref, allOf, oneOf/anyOf, type, enum, object properties
    with required/additionalProperties, and array items. Whole-string
    placeholders pass any schema. Errors are appended to `errors` with
    `loc` as the JSON-path-ish location prefix.
    """
    schema = _resolve_ref(schema, spec)
    if not isinstance(schema, dict) or not schema:
        return
    if isinstance(value, str) and _PLACEHOLDER_VALUE_RE.match(value):
        return

    if "allOf" in schema:
        # Shallow-merge the members into one object view so a property
        # declared in one member isn't reported unknown by another.
        merged = {k: v for k, v in schema.items() if k != "allOf"}
        for member in schema["allOf"]:
            member = _resolve_ref(member, spec)
            if not isinstance(member, dict):
                continue
            for key in ("type", "enum", "items", "additionalProperties"):
                if key in member and key not in merged:
                    merged[key] = member[key]
            if "properties" in member:
                merged.setdefault("properties", {}).update(member["properties"])
            if "required" in member:
                combined = merged.get("required", []) + member["required"]
                merged["required"] = list(dict.fromkeys(combined))
        _validate_body(value, merged, spec, loc, errors, check_required)
        return

    variants = schema.get("oneOf") or schema.get("anyOf")
    if variants:
        for variant in variants:
            variant_errors: list[str] = []
            _validate_body(value, variant, spec, loc, variant_errors, check_required)
            if not variant_errors:
                return
        errors.append(f"{loc}: does not match any schema variant accepted here")
        return

    t = schema.get("type")
    if t and not _type_ok(value, t):
        errors.append(f"{loc}: expected {t}, got {_json_type_name(value)}")
        return

    if "enum" in schema and value not in schema["enum"]:
        allowed = ", ".join(repr(x) for x in schema["enum"])
        errors.append(f"{loc}: {value!r} is not one of [{allowed}]")
        return

    if isinstance(value, dict):
        props = schema.get("properties")
        if isinstance(props, dict):
            additional = schema.get("additionalProperties")
            # JSON Schema treats an absent additionalProperties as "allow
            # anything"; we deliberately flag unknown keys anyway. The
            # specs document every accepted field for these endpoints,
            # and a typo'd field name the API would silently ignore is
            # exactly the bug this validator exists to catch. Only an
            # explicit additionalProperties (true or a schema) relaxes
            # the check.
            if not (additional is True or isinstance(additional, dict)):
                for key in value:
                    if key not in props:
                        errors.append(
                            f"{loc}: unknown property '{key}' "
                            f"(documented: {', '.join(sorted(props))})"
                        )
            for key, val in value.items():
                if key in props:
                    _validate_body(
                        val, props[key], spec, f"{loc}.{key}", errors, check_required
                    )
        if check_required:
            for req in schema.get("required", []):
                if req in value:
                    continue
                # readOnly properties are response-only; their presence in
                # `required` does not apply to request bodies.
                req_schema = {}
                if isinstance(props, dict) and req in props:
                    req_schema = _resolve_ref(props[req], spec)
                if isinstance(req_schema, dict) and req_schema.get("readOnly"):
                    continue
                errors.append(f"{loc}: missing required property '{req}'")
    elif isinstance(value, list):
        items = schema.get("items")
        if items:
            for i, val in enumerate(value):
                _validate_body(val, items, spec, f"{loc}[{i}]", errors, check_required)


def _parse_body_json(body: str):
    """Parse a curl payload as JSON. Returns (value, error_message)."""
    try:
        return json.loads(body), None
    except json.JSONDecodeError:
        pass
    # Retry with bare placeholders quoted, e.g. {"max_age": <seconds>}.
    try:
        return json.loads(_BARE_PLACEHOLDER_RE.sub(r'"<\1>"', body)), None
    except json.JSONDecodeError as e:
        return None, f"request body is not valid JSON ({e.msg})"


def _request_body_schema(spec: dict, tmpl: str, method: str):
    """Return (schema, required) for an operation's JSON request body.

    schema is None when the spec defines no JSON request body for the
    operation. Handles both OpenAPI 3 (requestBody) and Swagger 2.0
    (parameters with in: body); Swagger formData parameters describe
    form-encoded payloads, which we don't validate.
    """
    op = spec.get("paths", {}).get(tmpl, {}).get(method.lower(), {})
    request_body = _resolve_ref(op.get("requestBody"), spec)
    if isinstance(request_body, dict) and request_body:
        schema = (
            request_body.get("content", {})
            .get("application/json", {})
            .get("schema")
        )
        return schema, bool(request_body.get("required"))

    for param in op.get("parameters", []):  # Swagger 2.0
        param = _resolve_ref(param, spec)
        if isinstance(param, dict) and param.get("in") == "body":
            return param.get("schema"), bool(param.get("required"))

    return None, False


# ---------------------------------------------------------------------------
# Cloudflare quirks
# ---------------------------------------------------------------------------

def _cf_setting_request_schema(setting_id: str, components: dict) -> dict | None:
    """Return the narrowed request-body schema for a zone setting id, or
    None when the spec has no per-setting component (an unknown setting).

    Most settings follow the `zones_<setting_id>_value` convention with a
    {"value": ...} body; a few embed a service prefix in the component
    name (`zones_cache-rules_aegis_value` for `settings/aegis`), and
    ssl_recommender uses an `_enabled` component with an
    {"enabled": ...} body instead.
    """
    prop = "value"
    name = f"zones_{setting_id}_value"
    if name not in components:
        suffix = f"_{setting_id}_value"
        matches = sorted(
            n for n in components
            if n.startswith("zones_") and n.endswith(suffix)
        )
        if matches:
            # Unambiguous in the pinned spec; sorted() keeps the pick
            # deterministic should a future spec bump introduce a collision.
            name = matches[0]
        elif f"zones_{setting_id}_enabled" in components:
            prop = "enabled"
            name = f"zones_{setting_id}_enabled"
        else:
            return None
    return {
        "type": "object",
        "properties": {prop: {"$ref": f"#/components/schemas/{name}"}},
        "required": [prop],
    }


def _cloudflare_path_hook(
    matched: str, path: str, spec: dict
) -> tuple[dict | None, str | None]:
    """Cloudflare-specific path handling.

    The `/zones/{zone_id}/settings/{setting_id}` template matches any
    setting id, and its request schema is a oneOf over every zone
    setting — so a misspelled setting id would pass path validation and
    a wrong enum value could match some other setting's schema. When the
    setting id in the URL is a literal, resolve the per-setting
    component so both mistakes are caught.

    Returns (narrowed_body_schema, error). A non-None error fails the
    call; (None, None) means no quirk applies.
    """
    if matched != "/zones/{zone_id}/settings/{setting_id}":
        return None, None
    last_segment = path.rsplit("/", 1)[-1]
    if _PLACEHOLDER_SEGMENT_RE.match(last_segment):
        return None, None
    components = spec.get("components", {}).get("schemas", {})
    schema = _cf_setting_request_schema(last_segment, components)
    if schema is None:
        return None, f"unknown zone setting '{last_segment}'"
    return schema, None


# ---------------------------------------------------------------------------
# Provider registry
# ---------------------------------------------------------------------------
#
# Each provider entry:
#   policies        — policy files (relative to content/) to scan
#   host            — API host prefix curl calls are matched against
#   specs           — spec sources (see _load_spec); a provider may span
#                     multiple specs mounted at different base paths
#                     (Atlassian's org and user-management APIs)
#   body_exemptions — known divergences between a vendor's spec and the
#                     documented behavior of its API. Maps
#                     (METHOD, path_template) to the validation step to
#                     skip: "body" skips request-body validation
#                     entirely; "required" validates the body but skips
#                     required-property checks.
#   path_hook       — optional callback for provider-specific path
#                     template handling (see _cloudflare_path_hook)

API_PROVIDERS = {
    "cloudflare": {
        "policies": ["mondoo-cloudflare-security.mql.yaml"],
        "host": "https://api.cloudflare.com",
        "specs": [{
            "name": "cloudflare",
            "url": (
                "https://raw.githubusercontent.com/cloudflare/api-schemas/"
                f"{CLOUDFLARE_OPENAPI_SHA}/openapi.json"
            ),
            "pin": CLOUDFLARE_OPENAPI_SHA,
        }],
        "body_exemptions": {
            # The token-roll endpoint declares its body as `type: object`,
            # but the Cloudflare API docs (and the live API) require the
            # literal JSON string "" as the request body.
            ("PUT", "/user/tokens/{token_id}/value"): "body",
            # Account update reuses the shared account schema, which marks
            # `id` and `type` required because responses always carry
            # them; the documented update payload sends only `name` (plus
            # optional `settings`).
            ("PUT", "/accounts/{account_id}"): "required",
        },
        "path_hook": _cloudflare_path_hook,
    },
    "tailscale": {
        "policies": ["mondoo-tailscale-security.mql.yaml"],
        "host": "https://api.tailscale.com",
        # Tailscale serves its spec from a live endpoint
        # (https://api.tailscale.com/api/v2?openapi) that it documents as
        # unstable, so the spec is checked in via dump_api_specs.py.
        "specs": [{"name": "tailscale", "file": "tailscale_openapi.json"}],
    },
    "slack": {
        "policies": ["mondoo-slack-security.mql.yaml"],
        "host": "https://slack.com",
        "specs": [{
            "name": "slack",
            "url": (
                "https://raw.githubusercontent.com/slackapi/slack-api-specs/"
                f"{SLACK_OPENAPI_SHA}/web-api/slack_web_openapi_v2.json"
            ),
            "pin": SLACK_OPENAPI_SHA,
        }],
    },
    "atlassian": {
        "policies": ["mondoo-atlassian-security.mql.yaml"],
        "host": "https://api.atlassian.com",
        # Atlassian serves its admin API specs from dac-static.atlassian.com
        # with no version pinning, so both specs are checked in via
        # dump_api_specs.py. The org spec mounts at /admin, the
        # user-management spec at the API root.
        "specs": [
            {"name": "atlassian-org", "file": "atlassian_org_openapi.json"},
            {"name": "atlassian-um", "file": "atlassian_user_management_openapi.json"},
        ],
    },
    "grafana": {
        # Grafana is self-hosted; the policy's examples use the
        # documentation placeholder host below.
        "policies": ["mondoo-grafana-security.mql.yaml"],
        "host": "https://grafana.example.com",
        "specs": [{
            "name": "grafana",
            "url": (
                "https://raw.githubusercontent.com/grafana/grafana/"
                f"{GRAFANA_OPENAPI_SHA}/public/openapi3.json"
            ),
            "pin": GRAFANA_OPENAPI_SHA,
        }],
    },
    "vercel": {
        # The Vercel policy documents its non-CLI fixes as `curl` calls
        # against the REST API under `- id: api`, and its audit steps use
        # the same API, so both are validated (remediation_ids + the
        # default include_audit). Vercel serves its spec from a live,
        # unversioned endpoint (https://openapi.vercel.sh), so it is
        # checked in via dump_api_specs.py.
        #
        # strip_api_version: Vercel versions every path in the URL and
        # keeps multiple versions live, but the spec documents one version
        # per operation; normalizing the `/vN` prefix on both sides is what
        # makes the policy's `/v5/user/tokens` match the spec's `/v6`.
        "policies": ["mondoo-vercel-security.mql.yaml"],
        "host": "https://api.vercel.com",
        "specs": [{"name": "vercel", "file": "vercel_openapi.json"}],
        "remediation_ids": ("cli", "api"),
        "strip_api_version": True,
        # The list-stores endpoint is live but undocumented in the spec,
        # which carries only `/storage/stores/{id}`.
        "path_exemptions": {"/storage/stores"},
    },
}


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

def validate_api_curl(
    method: str,
    path: str,
    body: str | None,
    provider: dict,
    loaded_specs: list[tuple[dict, str, dict[str, set[str]]]],
) -> tuple[bool, list[str]]:
    """Validate one parsed curl call against a provider's specs.

    loaded_specs is [(spec_doc, mount, paths_index), ...] in registry
    order; the best (most literal) template match across all specs wins.
    """
    errors: list[str] = []

    if provider.get("strip_api_version"):
        path = _strip_api_version(path)

    # Endpoints the vendor's API serves but omits from its published spec
    # (e.g. Vercel's `GET /storage/stores` list — the spec documents only
    # `/storage/stores/{id}`). Compared version-stripped, matching how they
    # are written in the registry.
    if path in provider.get("path_exemptions", set()):
        return True, errors

    matched: str | None = None
    matched_spec: dict | None = None
    matched_index: dict[str, set[str]] | None = None
    matched_sub_path = path
    best_specificity = -1
    for spec, mount, paths_index in loaded_specs:
        if mount and not path.startswith(mount + "/"):
            continue
        sub_path = path[len(mount):] if mount else path
        tmpl = _match_spec_path(sub_path, paths_index)
        if tmpl is None:
            continue
        specificity = _template_specificity(tmpl)
        if specificity > best_specificity:
            matched = tmpl
            matched_spec = spec
            matched_index = paths_index
            matched_sub_path = sub_path
            best_specificity = specificity

    if matched is None or matched_spec is None or matched_index is None:
        errors.append(f"unknown API path '{path}'")
        return False, errors

    if method not in matched_index[matched]:
        allowed = sorted(matched_index[matched])
        errors.append(
            f"method '{method}' not supported on '{matched}' "
            f"(supported: {', '.join(allowed)})"
        )
        return False, errors

    narrowed_schema = None
    path_hook = provider.get("path_hook")
    if path_hook is not None:
        narrowed_schema, hook_error = path_hook(matched, matched_sub_path, matched_spec)
        if hook_error is not None:
            errors.append(hook_error)
            return False, errors

    exemption = provider.get("body_exemptions", {}).get((method, matched))
    if exemption == "body":
        return True, errors

    schema, body_required = _request_body_schema(matched_spec, matched, method)

    if body is None:
        if schema is not None and body_required:
            errors.append(
                f"'{method} {matched}' requires a request body, "
                "but the command sends none"
            )
            return False, errors
        return True, errors

    if body.startswith("@"):
        # Payload is read from a file the snippet builds; nothing to inspect.
        return True, errors

    if schema is None:
        # The operation takes no JSON body (or, for Swagger 2.0, takes
        # form-encoded parameters we don't validate).
        return True, errors

    value, parse_error = _parse_body_json(body)
    if parse_error:
        errors.append(parse_error)
        return False, errors

    if narrowed_schema is not None:
        schema = narrowed_schema

    _validate_body(
        value, schema, matched_spec, "body", errors,
        check_required=(exemption != "required"),
    )
    return (not errors), errors


def validate_api_provider(key: str) -> tuple[int, int]:
    """Validate one provider's API curl commands. Returns (pass, fail)."""
    provider = API_PROVIDERS[key]
    host = provider["host"]

    pass_count = 0
    fail_count = 0
    loaded_specs: list[tuple[dict, str, dict[str, set[str]]]] | None = None

    for policy_name in provider["policies"]:
        policy_file = SCRIPT_DIR / ".." / policy_name
        if not policy_file.exists():
            print(f"Error: Policy file not found: {policy_file}", file=sys.stderr)
            sys.exit(1)

        content = policy_file.read_text()
        # For API-first products the audit path is also a curl call, so
        # audit blocks are validated alongside remediation blocks. A
        # provider may document its fix under `- id: api` instead of the
        # default `- id: cli` (see remediation_ids).
        blocks = extract_bash_blocks(
            content,
            include_audit=True,
            remediation_ids=provider.get("remediation_ids", ("cli",)),
        )

        # Skip loading specs entirely if the policy has no curl calls
        # against this provider's API.
        if not any(host in b for b, _, _ in blocks):
            continue

        if loaded_specs is None:
            loaded_specs = []
            for source in provider["specs"]:
                spec = _load_spec(source)
                if provider.get("strip_api_version"):
                    _merge_version_stripped_paths(spec)
                loaded_specs.append((spec, _spec_mount(spec), _index_spec_paths(spec)))

        relpath = policy_relpath(policy_file)

        for block_text, block_line, uid in blocks:
            commands = split_commands(block_text, "curl", block_line)
            for cmd, line_num in commands:
                parsed = parse_api_curl(cmd, host)
                if parsed is None:
                    continue
                method, path, body = parsed

                is_valid, errors = validate_api_curl(
                    method, path, body, provider, loaded_specs
                )

                if is_valid:
                    print(f"[PASS] {uid}")
                    print(f"       {method} {path}")
                    pass_count += 1
                else:
                    print(f"[FAIL] {uid}")
                    print(f"       {method} {path}")
                    for error in errors:
                        print(f"       {error}")
                    fail_count += 1
                    FAILURES.append({
                        "file": relpath,
                        "line": line_num,
                        "uid": uid,
                        "command": f"{method} {path}",
                        "errors": errors,
                        "cloud": key,
                    })

    return pass_count, fail_count
