resource "openstack_sharedfilesystem_share_access_v2" "internal" {
  share_id     = openstack_sharedfilesystem_share_v2.data.id
  access_type  = "ip"
  access_to    = "10.0.0.0/8"
  access_level = "rw"
}
