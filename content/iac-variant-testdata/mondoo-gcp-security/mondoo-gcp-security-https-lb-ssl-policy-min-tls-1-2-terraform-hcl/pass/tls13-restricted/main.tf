# Compliant: HTTPS proxy references an SSL policy with RESTRICTED profile and TLS 1.3.
resource "google_compute_ssl_policy" "restricted" {
  name            = "restricted-ssl-policy"
  profile         = "RESTRICTED"
  min_tls_version = "TLS_1_3"
}

resource "google_compute_target_https_proxy" "default" {
  name             = "https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_ssl_certificate.default.id]
  ssl_policy       = google_compute_ssl_policy.restricted.id
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
