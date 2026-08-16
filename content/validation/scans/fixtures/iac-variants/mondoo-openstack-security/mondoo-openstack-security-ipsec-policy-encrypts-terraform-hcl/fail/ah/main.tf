# AH authenticates the payload but does not encrypt it, so traffic crosses the
# tunnel in cleartext.
resource "openstack_vpnaas_ipsec_policy_v2" "site_to_site" {
  name               = "site-to-site"
  transform_protocol = "ah"
  auth_algorithm     = "sha256"
  pfs                = "group14"
}
