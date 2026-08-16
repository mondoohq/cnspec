# Compliant: allow_open is false, so plaintext connections are rejected.
resource "google_network_security_server_tls_policy" "pass_example" {
  name       = "my-tls-policy"
  location   = "global"
  allow_open = false

  server_certificate {
    certificate_provider_instance {
      plugin_instance = "google_cloud_private_spiffe"
    }
  }
}
