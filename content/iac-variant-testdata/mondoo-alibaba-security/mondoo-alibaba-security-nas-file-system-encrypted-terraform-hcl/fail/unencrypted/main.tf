# encrypt_type 0 is no encryption, and it cannot be changed after creation, so
# this file system stores its contents in the clear for its whole life.
resource "alicloud_nas_file_system" "shared" {
  protocol_type = "NFS"
  storage_type  = "Performance"
  zone_id       = "cn-hangzhou-b"
  encrypt_type  = 0
}
