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
