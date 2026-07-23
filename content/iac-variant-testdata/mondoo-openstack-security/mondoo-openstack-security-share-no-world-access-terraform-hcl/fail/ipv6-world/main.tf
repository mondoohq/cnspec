resource "openstack_sharedfilesystem_share_access_v2" "public_v6" {
  share_id     = openstack_sharedfilesystem_share_v2.data.id
  access_type  = "ip"
  access_to    = "::/0"
  access_level = "rw"
}
