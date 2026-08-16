# Sessions the firewall cannot validate are decrypted and forwarded anyway: an expired
# certificate, an untrusted issuer, or a cipher suite the firewall does not support all
# pass straight through to the backend.
resource "oci_network_firewall_network_firewall_policy_decryption_profile" "forward_proxy" {
  name                                 = "forward-proxy"
  network_firewall_policy_id           = oci_network_firewall_network_firewall_policy.main.id
  type                                 = "SSL_FORWARD_PROXY"
  is_expired_certificate_blocked       = false
  is_untrusted_issuer_blocked          = false
  is_revocation_status_timeout_blocked = false
  is_unknown_revocation_status_blocked = false
  is_unsupported_cipher_blocked        = false
  is_unsupported_version_blocked       = false
}
