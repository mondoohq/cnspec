# Every validation failure ends the session rather than being forwarded.
resource "oci_network_firewall_network_firewall_policy_decryption_profile" "forward_proxy" {
  name                                 = "forward-proxy"
  network_firewall_policy_id           = oci_network_firewall_network_firewall_policy.main.id
  type                                 = "SSL_FORWARD_PROXY"
  is_expired_certificate_blocked       = true
  is_untrusted_issuer_blocked          = true
  is_revocation_status_timeout_blocked = true
  is_unknown_revocation_status_blocked = true
  is_unsupported_cipher_blocked        = true
  is_unsupported_version_blocked       = true
  is_out_of_capacity_blocked           = true
}
