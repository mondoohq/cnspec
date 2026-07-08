# HTTP listener without an http block: redirect_http defaults to false, so plain
# HTTP traffic is served and never redirected to HTTPS. This is insecure but the
# check passes vacuously because blocks.where(type=="http").all(...) is true on
# an empty set. BROKEN MQL — see findings.
resource "hcloud_load_balancer_service" "http" {
  load_balancer_id = hcloud_load_balancer.example.id
  protocol         = "http"
  listen_port      = 80
  destination_port = 80
}
