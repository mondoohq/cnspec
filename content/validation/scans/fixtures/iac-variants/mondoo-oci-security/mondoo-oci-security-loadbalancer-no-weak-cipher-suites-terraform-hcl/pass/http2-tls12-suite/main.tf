# Compliant: HTTP2 listener using a named TLS 1.2 cipher suite.
resource "oci_load_balancer_listener" "https" {
  load_balancer_id         = oci_load_balancer_load_balancer.lb.id
  name                     = "https"
  default_backend_set_name = "web-backends"
  port                     = 443
  protocol                 = "HTTP2"

  ssl_configuration {
    certificate_name  = "web-cert"
    cipher_suite_name = "oci-tls-1-2-2017"
    protocols         = ["TLSv1.2"]
  }
}
