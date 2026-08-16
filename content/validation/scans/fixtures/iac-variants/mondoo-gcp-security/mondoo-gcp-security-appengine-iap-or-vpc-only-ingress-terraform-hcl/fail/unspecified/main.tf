# Non-compliant: ingress traffic setting is unspecified (defaults to all).
resource "google_app_engine_service_network_settings" "unspecified" {
  service = "my-service"

  network_settings {
    ingress_traffic_allowed = "INGRESS_TRAFFIC_ALLOWED_UNSPECIFIED"
  }
}
