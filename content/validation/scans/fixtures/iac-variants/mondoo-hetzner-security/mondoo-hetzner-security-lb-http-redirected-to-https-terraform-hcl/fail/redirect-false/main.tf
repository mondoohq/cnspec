resource "hcloud_load_balancer_service" "http" {
  load_balancer_id = hcloud_load_balancer.example.id
  protocol         = "http"
  listen_port      = 80
  destination_port = 80

  http {
    redirect_http = false
  }
}
