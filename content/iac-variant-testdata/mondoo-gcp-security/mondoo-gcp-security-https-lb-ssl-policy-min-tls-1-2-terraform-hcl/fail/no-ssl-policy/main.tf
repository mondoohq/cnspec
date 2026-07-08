# Non-compliant: HTTPS proxy has no ssl_policy, so it uses the insecure default.
resource "google_compute_target_https_proxy" "default" {
  name             = "https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_ssl_certificate.default.id]
}

resource "google_compute_url_map" "default" {
  name            = "url-map"
  default_service = "backend"
}

resource "google_compute_ssl_certificate" "default" {
  name        = "cert"
  private_key = "key"
  certificate = "cert"
}
