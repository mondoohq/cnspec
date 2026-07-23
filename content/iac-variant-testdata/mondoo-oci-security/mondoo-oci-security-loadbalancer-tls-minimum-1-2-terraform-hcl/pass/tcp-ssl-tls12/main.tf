# Compliant: a TCP listener that terminates TLS via ssl_configuration offering
# only TLS 1.2. OCI terminates TLS on TCP listeners too (no "HTTPS" protocol),
# so the check must still evaluate its TLS versions.
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
