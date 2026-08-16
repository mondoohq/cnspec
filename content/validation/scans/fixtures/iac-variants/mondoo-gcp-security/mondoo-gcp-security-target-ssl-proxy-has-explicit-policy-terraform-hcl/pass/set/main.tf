resource "google_compute_ssl_policy" "custom" {
  name    = "custom-ssl-policy"
  profile = "MODERN"
}

resource "google_compute_target_ssl_proxy" "pass" {
  name             = "test-proxy"
  backend_service  = google_compute_backend_service.default.id
  ssl_certificates = [google_compute_ssl_certificate.default.id]
  ssl_policy       = google_compute_ssl_policy.custom.id
}
