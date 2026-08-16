resource "google_compute_ssl_policy" "restricted" {
  name            = "restricted-ssl-policy"
  profile         = "RESTRICTED"
  min_tls_version = "TLS_1_2"
}
