# Single DES has a 56-bit key and is brute-forceable; the IPsec policy governs
# the data channel, so payload traffic is effectively unprotected.
resource "openstack_vpnaas_ipsec_policy_v2" "site_to_site" {
  name                 = "site-to-site-ipsec"
  transform_protocol   = "esp"
  encryption_algorithm = "des"
  auth_algorithm       = "sha1"
  pfs                  = "group5"
}
