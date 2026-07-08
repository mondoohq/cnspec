# Non-compliant: min_tls_version is unset, defaulting to TLS 1.0.
resource "google_compute_ssl_policy" "fail_example" {
  name    = "my-ssl-policy"
  profile = "MODERN"
}
