resource "stackit_sfs_resource_pool" "shared" {
  project_id        = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name              = "shared"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["10.0.0.0/8"]

}
