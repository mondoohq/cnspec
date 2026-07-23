resource "openstack_images_image_v2" "golden" {
  name             = "golden-base"
  local_file_path  = "/images/golden-base.qcow2"
  container_format = "bare"
  disk_format      = "qcow2"
  visibility       = "shared"
}
