# Compliant: ingress restricted to internal traffic and the load balancer.
resource "google_app_engine_service_network_settings" "internal_lb" {
  service = "my-service"

  network_settings {
    ingress_traffic_allowed = "INGRESS_TRAFFIC_ALLOWED_INTERNAL_AND_LB"
  }
}
