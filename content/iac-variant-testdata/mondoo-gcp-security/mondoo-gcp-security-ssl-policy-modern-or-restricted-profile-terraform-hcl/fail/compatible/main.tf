resource "google_compute_ssl_policy" "compatible" {
  name    = "compatible-ssl-policy"
  profile = "COMPATIBLE"
}
