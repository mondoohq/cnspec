# A plain HTTP listener serves traffic unencrypted. The check fails on any
# service with protocol http, whether or not it has an http block.
resource "hcloud_load_balancer_service" "http" {
  load_balancer_id = hcloud_load_balancer.example.id
  protocol         = "http"
  listen_port      = 80
  destination_port = 80
}
