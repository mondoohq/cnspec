# Non-compliant: all ingress traffic is allowed.
resource "google_app_engine_service_network_settings" "public" {
  service = "default"

  network_settings {
    ingress_traffic_allowed = "INGRESS_TRAFFIC_ALLOWED_ALL"
  }
}
