resource "google_compute_backend_service" "fail" {
  name        = "cdn-backend"
  protocol    = "HTTPS"
  enable_cdn  = true
  timeout_sec = 30
}
