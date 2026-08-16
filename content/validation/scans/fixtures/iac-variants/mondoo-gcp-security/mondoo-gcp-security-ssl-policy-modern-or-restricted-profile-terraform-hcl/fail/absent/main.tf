resource "google_compute_ssl_policy" "no_profile" {
  name            = "default-ssl-policy"
  min_tls_version = "TLS_1_1"
}
