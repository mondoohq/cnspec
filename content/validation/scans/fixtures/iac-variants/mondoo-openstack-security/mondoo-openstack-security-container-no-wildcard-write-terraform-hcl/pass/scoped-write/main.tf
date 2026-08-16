resource "openstack_objectstorage_container_v1" "backups" {
  name          = "prod-backups"
  region        = "RegionOne"
  container_read  = "project:backup-reader"
  container_write = "project:backup-writer"

  metadata = {
    environment = "production"
  }
}
