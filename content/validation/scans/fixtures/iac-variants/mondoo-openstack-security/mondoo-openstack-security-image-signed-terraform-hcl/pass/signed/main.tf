resource "openstack_images_image_v2" "ubuntu" {
  name             = "ubuntu-24.04"
  local_file_path  = "/images/ubuntu-24.04.qcow2"
  container_format = "bare"
  disk_format      = "qcow2"
  visibility       = "private"

  properties = {
    img_signature            = "iRfL5C3...base64signature...=="
    img_signature_hash_method = "SHA-256"
    img_signature_key_type    = "RSA-PSS"
    img_signature_certificate_uuid = "0d9e0e50-3f1e-4b1a-9a1e-2c3d4e5f6a7b"
  }
}
