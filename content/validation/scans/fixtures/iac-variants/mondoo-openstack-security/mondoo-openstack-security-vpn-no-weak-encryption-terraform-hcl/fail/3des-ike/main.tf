# 3DES is a 64-bit block cipher deprecated by NIST; the IKE policy negotiates
# the tunnel keys, so a weak cipher here undermines the whole VPN.
resource "openstack_vpnaas_ike_policy_v2" "site_to_site" {
  name                 = "site-to-site-ike"
  encryption_algorithm = "3des"
  auth_algorithm       = "sha1"
  pfs                  = "group5"
}
