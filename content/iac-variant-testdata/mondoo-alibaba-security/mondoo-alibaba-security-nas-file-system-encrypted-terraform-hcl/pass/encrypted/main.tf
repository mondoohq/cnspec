# The file system is encrypted at rest.
resource "alicloud_nas_file_system" "shared" {
  protocol_type = "NFS"
  storage_type  = "Performance"
  zone_id       = "cn-hangzhou-b"
  encrypt_type  = 1
}
