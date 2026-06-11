# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# REST API curl validation against vendor OpenAPI specs.

import json
import re
import sys
import urllib.error
import urllib.request

from pathlib import Path

from .common import FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands


# ---------------------------------------------------------------------------
# Cloudflare API validation
# ---------------------------------------------------------------------------
#
# Cloudflare's `flarectl` is read-only for zone settings and `wrangler` does
# not cover the surface area this policy targets (SSL/TLS, WAF, security
# level, 2FA enforcement, etc.). The realistic CLI path is `curl` against
# the Cloudflare REST API, so the validator scans for those calls and
# verifies each path + HTTP method — and, where the call sends a JSON
# payload, the request body — against the official OpenAPI spec published
# at https://github.com/cloudflare/api-schemas.
#
# Body validation covers the subset of JSON Schema the Cloudflare spec
# actually uses for the endpoints in this policy: $ref, allOf, oneOf/anyOf,
# type, enum, properties/required, and array items. Angle-bracket
# placeholders (`"<account-name>"`) act as wildcards that satisfy any
# schema. For the generic `/zones/{zone_id}/settings/{setting_id}` PATCH —
# whose spec body is a oneOf over every zone setting — the validator
# resolves the per-setting component (`zones_<setting_id>_value`) from the
# literal setting id in the URL, so wrong enum values and misspelled
# payload fields fail instead of slipping through the union.
#
# The spec is large (~9 MB) and the upstream repo doesn't ship releases —
# `main` is the only branch and it's auto-updated. We pin a known-good
# commit SHA so validation is deterministic; bumping CLOUDFLARE_OPENAPI_SHA
# is a deliberate maintainer action, like bumping doctl or the AWS CLI
# version. The fetched spec is cached on disk under ~/.cache so we don't
# re-download on every run.

CLOUDFLARE_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-cloudflare-security.mql.yaml"

CLOUDFLARE_OPENAPI_SHA = "a0e7cfa11b0d08b05a0b373f47ea722bd48ca7c4"
CLOUDFLARE_OPENAPI_URL = (
    f"https://raw.githubusercontent.com/cloudflare/api-schemas/"
    f"{CLOUDFLARE_OPENAPI_SHA}/openapi.json"
)
CLOUDFLARE_API_PREFIX = "https://api.cloudflare.com/client/v4"


def _cloudflare_cache_path() -> Path:
    cache_dir = Path.home() / ".cache" / "cnspec-validation"
    cache_dir.mkdir(parents=True, exist_ok=True)
    return cache_dir / f"cloudflare-openapi-{CLOUDFLARE_OPENAPI_SHA[:12]}.json"


def _load_cloudflare_openapi() -> dict:
    """Return the parsed Cloudflare OpenAPI spec, downloading and caching
    on first use."""
    cache = _cloudflare_cache_path()
    if not cache.exists():
        print(
            f"Fetching Cloudflare OpenAPI spec (sha {CLOUDFLARE_OPENAPI_SHA[:12]})...",
            file=sys.stderr,
        )
        try:
            with urllib.request.urlopen(CLOUDFLARE_OPENAPI_URL, timeout=60) as r:
                cache.write_bytes(r.read())
        except (urllib.error.URLError, TimeoutError) as e:
            print(
                f"Error: failed to download Cloudflare OpenAPI spec from\n"
                f"  {CLOUDFLARE_OPENAPI_URL}\n"
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
            f"Error: cached Cloudflare OpenAPI spec is corrupted\n"
            f"  ({cache}: {e})\n"
            "\n"
            "Delete the cache file and re-run to re-download:\n"
            f"  rm {cache}",
            file=sys.stderr,
        )
        sys.exit(1)


def _index_cloudflare_paths(spec: dict) -> dict[str, set[str]]:
    """Index OpenAPI paths so we can match a concrete URL path to a path
    template and verify the HTTP method.

    Returns {path_template: {http_method, ...}} where http_method is the
    upper-cased verb. Path templates are stored verbatim from the spec
    (e.g. "/zones/{zone_id}/settings/{setting_id}").
    """
    out: dict[str, set[str]] = {}
    for path, ops in spec.get("paths", {}).items():
        methods = {m.upper() for m in ops if m in (
            "get", "post", "put", "patch", "delete", "head", "options",
        )}
        if methods:
            out[path] = methods
    return out


# Placeholder tokens used in remediation examples (e.g. <zone-id>,
# <account-id>). The validator treats these as wildcards that match any
# OpenAPI path parameter ({zone_id}, {account_id}, ...).
_CF_PLACEHOLDER_RE = re.compile(r"^<[a-z][a-z0-9-]*>$")


def _match_cloudflare_path(curl_path: str, openapi_paths: dict[str, set[str]]) -> str | None:
    """Find an OpenAPI path template matching a concrete curl URL path.

    A segment matches if:
      - both segments are literally equal, or
      - the OpenAPI segment is a `{param}` template and the curl segment
        is either a placeholder like `<zone-id>` or a concrete literal.

    When multiple templates match, prefers the one with the most literal
    (non-`{param}`) segments — so e.g. `/zones/{id}/dns_records/import`
    wins over `/zones/{id}/dns_records/{record_id}` for a curl call to
    `/zones/<zone-id>/dns_records/import`. Returns the matched path
    template, or None.
    """
    curl_parts = curl_path.split("/")

    # Exact-literal match first.
    if curl_path in openapi_paths:
        return curl_path

    best: str | None = None
    best_specificity = -1
    for tmpl in openapi_paths:
        tmpl_parts = tmpl.split("/")
        if len(tmpl_parts) != len(curl_parts):
            continue
        if not all(_segment_matches(t, c) for t, c in zip(tmpl_parts, curl_parts)):
            continue
        specificity = sum(
            1 for p in tmpl_parts if p and not (p.startswith("{") and p.endswith("}"))
        )
        if specificity > best_specificity:
            best = tmpl
            best_specificity = specificity
    return best


def _segment_matches(tmpl_seg: str, curl_seg: str) -> bool:
    if tmpl_seg == curl_seg:
        return True
    if tmpl_seg.startswith("{") and tmpl_seg.endswith("}"):
        # Templated segment matches any concrete value or angle-bracket
        # placeholder in the curl URL.
        return curl_seg != "" and "/" not in curl_seg
    return False


# Match a curl invocation. The URL may be quoted or unquoted; -X / --request
# may appear before or after it.
_CF_CURL_URL_RE = re.compile(
    r'https?://api\.cloudflare\.com/client/v4(/[^\s\'"]+)'
)
_CF_CURL_METHOD_RE = re.compile(
    r'(?:^|\s)(?:-X|--request)(?:\s+|=)([A-Z]+)'
)
# Any curl flag that supplies a request body. split_commands() has already
# shlex-tokenized and re-joined the command, so the payload appears after
# the flag with its shell quoting stripped but its JSON quoting intact.
_CF_CURL_DATA_RE = re.compile(
    r"(?:^|\s)(?:--data(?:-raw|-binary|-ascii)?|--json|-d)(?:\s+|=)"
)

# A whole-string placeholder value like "<account-name>". During body
# validation these act as wildcards that satisfy any schema.
_CF_PLACEHOLDER_VALUE_RE = re.compile(r"^<[A-Za-z0-9_.-]+>$")

# Quote a bare (unquoted) placeholder so the payload parses as JSON, e.g.
# {"max_age": <seconds>} -> {"max_age": "<seconds>"}. Placeholders already
# inside JSON strings are excluded by the quote look-around.
_CF_BARE_PLACEHOLDER_RE = re.compile(r'(?<![\w"<])<([A-Za-z0-9_.-]+)>(?![\w">])')

# Known divergences between Cloudflare's OpenAPI spec and the documented
# behavior of the API. Maps (METHOD, path_template) to the validation step
# to skip:
#   "body"     — skip request-body validation entirely.
#   "required" — validate the body but skip required-property checks.
CLOUDFLARE_BODY_EXEMPTIONS = {
    # The token-roll endpoint declares its body as `type: object`, but the
    # Cloudflare API docs (and the live API) require the literal JSON
    # string "" as the request body.
    ("PUT", "/user/tokens/{token_id}/value"): "body",
    # Account update reuses the shared account schema, which marks `id` and
    # `type` required because responses always carry them; the documented
    # update payload sends only `name` (plus optional `settings`).
    ("PUT", "/accounts/{account_id}"): "required",
}


def parse_cloudflare_curl(cmd: str) -> tuple[str, str, str | None] | None:
    """Extract (HTTP_METHOD, URL_PATH, BODY) from a curl Cloudflare API call.

    Returns None if the command is not a Cloudflare API curl. The method
    defaults to GET when -X / --request is absent (matches curl's default
    when no body is supplied; for POST/PATCH/PUT we always require an
    explicit -X in remediation snippets). BODY is the raw --data payload,
    or None when the command sends no body.
    """
    if "api.cloudflare.com/client/v4" not in cmd:
        return None
    if not cmd.lstrip().startswith("curl"):
        return None

    url_match = _CF_CURL_URL_RE.search(cmd)
    if not url_match:
        return None
    path = url_match.group(1)
    # Drop a fragment or query string if present.
    path = path.split("?", 1)[0].split("#", 1)[0]

    method_match = _CF_CURL_METHOD_RE.search(cmd)
    method = method_match.group(1).upper() if method_match else "GET"

    return method, path, _extract_curl_body(cmd)


def _extract_curl_body(cmd: str) -> str | None:
    """Return the payload passed via --data/-d/--json, or None.

    The command string has been shlex-tokenized and space-joined, so the
    payload's shell quotes are gone. A JSON object/array payload may
    therefore contain spaces; scan to its balanced closing brace instead of
    splitting on whitespace.
    """
    m = _CF_CURL_DATA_RE.search(cmd)
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


def _cf_resolve_ref(node, spec: dict):
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


def _cf_type_ok(value, t) -> bool:
    if isinstance(t, list):
        return any(_cf_type_ok(value, x) for x in t)
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


def _cf_json_type_name(value) -> str:
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


def _cf_validate_body(
    value,
    schema,
    spec: dict,
    loc: str,
    errors: list[str],
    check_required: bool = True,
) -> None:
    """Validate a parsed JSON value against a (subset of) JSON Schema.

    Covers what the Cloudflare spec uses for the request bodies in this
    policy: $ref, allOf, oneOf/anyOf, type, enum, object properties with
    required/additionalProperties, and array items. Whole-string
    placeholders pass any schema. Errors are appended to `errors` with
    `loc` as the JSON-path-ish location prefix.
    """
    schema = _cf_resolve_ref(schema, spec)
    if not isinstance(schema, dict) or not schema:
        return
    if isinstance(value, str) and _CF_PLACEHOLDER_VALUE_RE.match(value):
        return

    if "allOf" in schema:
        # Shallow-merge the members into one object view so a property
        # declared in one member isn't reported unknown by another.
        merged = {k: v for k, v in schema.items() if k != "allOf"}
        for member in schema["allOf"]:
            member = _cf_resolve_ref(member, spec)
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
        _cf_validate_body(value, merged, spec, loc, errors, check_required)
        return

    variants = schema.get("oneOf") or schema.get("anyOf")
    if variants:
        for variant in variants:
            variant_errors: list[str] = []
            _cf_validate_body(
                value, variant, spec, loc, variant_errors, check_required
            )
            if not variant_errors:
                return
        errors.append(f"{loc}: does not match any schema variant accepted here")
        return

    t = schema.get("type")
    if t and not _cf_type_ok(value, t):
        errors.append(f"{loc}: expected {t}, got {_cf_json_type_name(value)}")
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
            # anything"; we deliberately flag unknown keys anyway. The spec
            # documents every accepted field for these endpoints, and a
            # typo'd field name the API would silently ignore is exactly
            # the bug this validator exists to catch. Only an explicit
            # additionalProperties (true or a schema) relaxes the check.
            if not (additional is True or isinstance(additional, dict)):
                for key in value:
                    if key not in props:
                        errors.append(
                            f"{loc}: unknown property '{key}' "
                            f"(documented: {', '.join(sorted(props))})"
                        )
            for key, val in value.items():
                if key in props:
                    _cf_validate_body(
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
                    req_schema = _cf_resolve_ref(props[req], spec)
                if isinstance(req_schema, dict) and req_schema.get("readOnly"):
                    continue
                errors.append(f"{loc}: missing required property '{req}'")
    elif isinstance(value, list):
        items = schema.get("items")
        if items:
            for i, val in enumerate(value):
                _cf_validate_body(
                    val, items, spec, f"{loc}[{i}]", errors, check_required
                )


def _cf_parse_body_json(body: str):
    """Parse a curl payload as JSON. Returns (value, error_message)."""
    try:
        return json.loads(body), None
    except json.JSONDecodeError:
        pass
    # Retry with bare placeholders quoted, e.g. {"max_age": <seconds>}.
    try:
        return json.loads(_CF_BARE_PLACEHOLDER_RE.sub(r'"<\1>"', body)), None
    except json.JSONDecodeError as e:
        return None, f"request body is not valid JSON ({e.msg})"


def _cf_request_body_schema(spec: dict, tmpl: str, method: str):
    """Return (schema, required) for an operation's JSON request body.

    schema is None when the spec defines no request body for the operation.
    """
    op = spec.get("paths", {}).get(tmpl, {}).get(method.lower(), {})
    request_body = _cf_resolve_ref(op.get("requestBody"), spec)
    if not isinstance(request_body, dict) or not request_body:
        return None, False
    schema = (
        request_body.get("content", {})
        .get("application/json", {})
        .get("schema")
    )
    return schema, bool(request_body.get("required"))


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


def validate_cloudflare_curl(
    method: str,
    path: str,
    body: str | None,
    spec: dict,
    openapi_paths: dict[str, set[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed Cloudflare curl call against the OpenAPI spec."""
    errors: list[str] = []

    matched = _match_cloudflare_path(path, openapi_paths)
    if not matched:
        errors.append(f"unknown Cloudflare API path '{path}'")
        return False, errors

    if method not in openapi_paths[matched]:
        allowed = sorted(openapi_paths[matched])
        errors.append(
            f"method '{method}' not supported on '{matched}' "
            f"(supported: {', '.join(allowed)})"
        )
        return False, errors

    components = spec.get("components", {}).get("schemas", {})

    # The settings path template matches any {setting_id}, so a misspelled
    # setting id would otherwise pass path validation.
    setting_schema = None
    if matched == "/zones/{zone_id}/settings/{setting_id}":
        last_segment = path.rsplit("/", 1)[-1]
        if not _CF_PLACEHOLDER_RE.match(last_segment):
            setting_schema = _cf_setting_request_schema(last_segment, components)
            if setting_schema is None:
                errors.append(f"unknown zone setting '{last_segment}'")
                return False, errors

    exemption = CLOUDFLARE_BODY_EXEMPTIONS.get((method, matched))
    if exemption == "body":
        return True, errors

    schema, body_required = _cf_request_body_schema(spec, matched, method)

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
        errors.append(f"'{method} {matched}' does not accept a request body")
        return False, errors

    value, parse_error = _cf_parse_body_json(body)
    if parse_error:
        errors.append(parse_error)
        return False, errors

    # For the generic settings endpoint, narrow the oneOf-over-everything
    # request schema down to the specific setting named in the URL so enum
    # and field-name mistakes are actually caught.
    if setting_schema is not None:
        schema = setting_schema

    _cf_validate_body(
        value, schema, spec, "body", errors,
        check_required=(exemption != "required"),
    )
    return (not errors), errors


def validate_cloudflare() -> tuple[int, int]:
    """Validate Cloudflare API curl commands. Returns (pass_count, fail_count)."""
    if not CLOUDFLARE_POLICY_FILE.exists():
        print(
            f"Error: Policy file not found: {CLOUDFLARE_POLICY_FILE}",
            file=sys.stderr,
        )
        sys.exit(1)

    content = CLOUDFLARE_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    # Skip the OpenAPI download entirely if there are no Cloudflare API
    # curl calls in the policy.
    has_cf_curl = any(
        "api.cloudflare.com/client/v4" in b for b, _, _ in blocks
    )
    if not has_cf_curl:
        return 0, 0

    spec = _load_cloudflare_openapi()
    openapi_paths = _index_cloudflare_paths(spec)

    pass_count = 0
    fail_count = 0

    relpath = policy_relpath(CLOUDFLARE_POLICY_FILE)

    for block_text, block_line, uid in blocks:
        commands = split_commands(block_text, "curl", block_line)
        for cmd, line_num in commands:
            parsed = parse_cloudflare_curl(cmd)
            if parsed is None:
                continue
            method, path, body = parsed

            is_valid, errors = validate_cloudflare_curl(
                method, path, body, spec, openapi_paths
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
                    "cloud": "cloudflare",
                })

    return pass_count, fail_count
