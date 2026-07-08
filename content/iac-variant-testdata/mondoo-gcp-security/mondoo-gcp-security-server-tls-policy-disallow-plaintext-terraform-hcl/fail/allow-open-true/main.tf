# Non-compliant: allow_open is true, permitting plaintext connections.
resource "google_network_security_server_tls_policy" "fail_example" {
  name       = "my-tls-policy"
  location   = "global"
  allow_open = true
}
