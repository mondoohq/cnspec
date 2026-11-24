resource "google_compute_backend_service" "pass" {
  name        = "cdn-backend"
  protocol    = "HTTPS"
  enable_cdn  = true
  timeout_sec = 30

  cdn_policy {
    cache_mode                   = "CACHE_ALL_STATIC"
    signed_url_cache_max_age_sec = 3600
    default_ttl                  = 3600
  }
}

resource "google_compute_backend_service_signed_url_key" "pass" {
  name            = "signing-key"
  key_value       = var.signing_key_value
  backend_service = google_compute_backend_service.pass.name
}

variable "signing_key_value" {
  type      = string
  sensitive = true
}
