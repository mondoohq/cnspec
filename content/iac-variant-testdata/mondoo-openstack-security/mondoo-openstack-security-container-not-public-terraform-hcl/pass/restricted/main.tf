resource "openstack_objectstorage_container_v1" "backups" {
  name           = "backups"
  region         = "RegionOne"
  container_read = ".r:partner.example.com,.rlistings"
}
