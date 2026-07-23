# Non-compliant: SSL policy allows TLS 1.0, below the required minimum.
resource "google_compute_ssl_policy" "fail_example" {
  name            = "my-ssl-policy"
  profile         = "MODERN"
  min_tls_version = "TLS_1_0"
}
