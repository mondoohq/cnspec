# Compliant: SSL policy enforces a minimum TLS version of 1.3.
resource "google_compute_ssl_policy" "pass_example" {
  name            = "my-ssl-policy"
  profile         = "RESTRICTED"
  min_tls_version = "TLS_1_3"
}
