# Compliant: HTTP2 listener using the modern cipher suite.
resource "oci_load_balancer_listener" "https" {
  load_balancer_id         = oci_load_balancer_load_balancer.lb.id
  name                     = "https"
  default_backend_set_name = "web-backends"
  port                     = 443
  protocol                 = "HTTP2"

  ssl_configuration {
    certificate_name  = "web-cert"
    cipher_suite_name = "oci-modern-ssl-cipher-suite-v1"
    protocols         = ["TLSv1.2", "TLSv1.3"]
  }
}
