resource "hcloud_load_balancer_service" "https" {
  load_balancer_id = hcloud_load_balancer.example.id
  protocol         = "https"
  listen_port      = 443
  destination_port = 80

  http {
    sticky_sessions = true
  }
}
