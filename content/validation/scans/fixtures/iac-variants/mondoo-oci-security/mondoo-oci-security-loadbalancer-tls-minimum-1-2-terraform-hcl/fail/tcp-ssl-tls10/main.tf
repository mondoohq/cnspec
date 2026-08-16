# Non-compliant: a TCP listener terminating TLS via ssl_configuration that still
# offers legacy TLS 1.0. Previously skipped because protocol != HTTPS/HTTP2;
# now correctly flagged since it has an ssl_configuration block.
resource "oci_load_balancer_listener" "tcp_tls" {
  load_balancer_id         = oci_load_balancer_load_balancer.lb.id
  name                     = "tcp-tls"
  default_backend_set_name = "web-backends"
  port                     = 8443
  protocol                 = "TCP"

  ssl_configuration {
    certificate_name  = "web-cert"
    cipher_suite_name = "oci-modern-ssl-cipher-suite-v1"
    protocols         = ["TLSv1", "TLSv1.2"]
  }
}
