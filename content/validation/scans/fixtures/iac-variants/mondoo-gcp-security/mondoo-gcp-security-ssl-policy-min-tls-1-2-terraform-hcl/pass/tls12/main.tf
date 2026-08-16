# Compliant: SSL policy enforces a minimum TLS version of 1.2.
resource "google_compute_ssl_policy" "pass_example" {
  name            = "my-ssl-policy"
  profile         = "MODERN"
  min_tls_version = "TLS_1_2"
}
