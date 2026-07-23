# Compliant: allow_open is not set, defaulting to disallow plaintext.
resource "google_network_security_server_tls_policy" "pass_example" {
  name     = "my-tls-policy"
  location = "global"

  server_certificate {
    certificate_provider_instance {
      plugin_instance = "google_cloud_private_spiffe"
    }
  }
}
