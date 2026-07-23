# Compliant: ingress restricted to internal traffic only.
resource "google_app_engine_service_network_settings" "internal" {
  service = "default"

  network_settings {
    ingress_traffic_allowed = "INGRESS_TRAFFIC_ALLOWED_INTERNAL_ONLY"
  }
}
