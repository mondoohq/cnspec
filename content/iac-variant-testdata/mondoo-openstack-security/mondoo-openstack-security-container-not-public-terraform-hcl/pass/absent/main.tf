resource "openstack_objectstorage_container_v1" "backups" {
  name   = "backups"
  region = "RegionOne"

  metadata = {
    Owner = "platform-team"
  }
}
