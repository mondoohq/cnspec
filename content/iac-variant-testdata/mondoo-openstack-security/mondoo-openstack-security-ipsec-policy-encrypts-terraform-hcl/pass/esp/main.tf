resource "openstack_vpnaas_ipsec_policy_v2" "site_to_site" {
  name                 = "site-to-site"
  transform_protocol   = "esp"
  encryption_algorithm = "aes-256"
  auth_algorithm       = "sha256"
  pfs                  = "group14"
}
