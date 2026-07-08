# Compliant: TCP listener terminating TLS via ssl_configuration with a modern
# cipher suite. Must be evaluated even though protocol is not HTTPS/HTTP2.
resource "oci_load_balancer_listener" "tcp_tls" {
  load_balancer_id         = oci_load_balancer_load_balancer.lb.id
  name                     = "tcp-tls"
  default_backend_set_name = "web-backends"
  port                     = 8443
  protocol                 = "TCP"

  ssl_configuration {
    certificate_name  = "web-cert"
    cipher_suite_name = "oci-modern-ssl-cipher-suite-v1"
    protocols         = ["TLSv1.2"]
  }
}
