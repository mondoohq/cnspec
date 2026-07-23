resource "openstack_images_image_v2" "ubuntu" {
  name             = "ubuntu-24.04"
  image_source_url = "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
  container_format = "bare"
  disk_format      = "qcow2"
  visibility       = "private"
}
