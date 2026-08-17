resource "stackit_sfs_export_policy" "shared" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name       = "shared"
  rules = [
    {
      order  = 1
      ip_acl = ["::/0"]
    },
    {
      order  = 2
      ip_acl = ["172.16.0.0/24"]
    }
  ]
}
