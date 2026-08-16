resource "google_compute_ssl_policy" "custom" {
  name            = "custom-ssl-policy"
  profile         = "CUSTOM"
  min_tls_version = "TLS_1_0"
  custom_features = ["TLS_RSA_WITH_AES_128_CBC_SHA"]
}
