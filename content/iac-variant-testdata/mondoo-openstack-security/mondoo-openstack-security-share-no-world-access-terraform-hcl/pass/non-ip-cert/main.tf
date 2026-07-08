resource "openstack_sharedfilesystem_share_access_v2" "cert" {
  share_id     = openstack_sharedfilesystem_share_v2.data.id
  access_type  = "cert"
  access_to    = "tenant.example.com"
  access_level = "rw"
}
