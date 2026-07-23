resource "openstack_sharedfilesystem_share_v2" "data" {
  name             = "app-data"
  description      = "Shared data volume for the app tier"
  share_proto      = "NFS"
  size             = 10
  share_network_id = openstack_sharedfilesystem_sharenetwork_v2.net.id
  is_public        = true
}
