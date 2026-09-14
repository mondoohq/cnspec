resource "datadog_domain_allowlist" "main" {
  enabled = true
  domains = ["acme.com", "acme.org"]
}
