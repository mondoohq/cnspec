resource "stackit_service_account_key" "ci" {
  project_id            = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  service_account_email = "ci@sa.stackit.cloud"
  ttl_days              = 90
}
