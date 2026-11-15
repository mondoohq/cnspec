resource "google_compute_backend_service" "signed" {
  name       = "cdn-signed"
  protocol   = "HTTPS"
  enable_cdn = true
}

resource "google_compute_backend_service" "fail" {
  name       = "cdn-unsigned"
  protocol   = "HTTPS"
  enable_cdn = true
}

resource "google_compute_backend_service_signed_url_key" "signed" {
  name            = "signing-key"
  key_value       = var.signing_key_value
  backend_service = google_compute_backend_service.signed.name
}

variable "signing_key_value" {
  type      = string
  sensitive = true
}
