# Non-compliant: HTTP2 listener using the broad backwards-compatible cipher suite.
resource "oci_load_balancer_listener" "https" {
  load_balancer_id         = oci_load_balancer_load_balancer.lb.id
  name                     = "https"
  default_backend_set_name = "web-backends"
  port                     = 443
  protocol                 = "HTTP2"

  ssl_configuration {
    certificate_name  = "web-cert"
    cipher_suite_name = "oci-wider-compatible-ssl-cipher-suite-v1"
    protocols         = ["TLSv1.2"]
  }
}
