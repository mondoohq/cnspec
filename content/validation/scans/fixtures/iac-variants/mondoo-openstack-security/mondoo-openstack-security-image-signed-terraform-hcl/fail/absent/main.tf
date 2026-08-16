resource "openstack_images_image_v2" "ubuntu" {
  name             = "ubuntu-24.04"
  local_file_path  = "/images/ubuntu-24.04.qcow2"
  container_format = "bare"
  disk_format      = "qcow2"
  visibility       = "private"
}
