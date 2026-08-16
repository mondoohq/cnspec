# A "*" write ACL lets any authenticated tenant upload into the container.
resource "openstack_objectstorage_container_v1" "backups" {
  name            = "prod-backups"
  region          = "RegionOne"
  container_read  = ".r:*,.rlistings"
  container_write = "*:*"
}
