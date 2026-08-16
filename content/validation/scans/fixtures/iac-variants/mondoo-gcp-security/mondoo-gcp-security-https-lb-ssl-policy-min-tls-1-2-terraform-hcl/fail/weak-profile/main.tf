# Non-compliant: SSL policy uses the COMPATIBLE profile, which allows weak ciphers.
resource "google_compute_ssl_policy" "compatible" {
  name            = "compatible-ssl-policy"
  profile         = "COMPATIBLE"
  min_tls_version = "TLS_1_2"
}

resource "google_compute_target_https_proxy" "default" {
  name             = "https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_ssl_certificate.default.id]
  ssl_policy       = google_compute_ssl_policy.compatible.id
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
