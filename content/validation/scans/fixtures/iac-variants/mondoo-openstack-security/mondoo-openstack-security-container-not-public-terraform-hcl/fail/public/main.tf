resource "openstack_objectstorage_container_v1" "assets" {
  name           = "public-assets"
  region         = "RegionOne"
  container_read = ".r:*,.rlistings"
}
