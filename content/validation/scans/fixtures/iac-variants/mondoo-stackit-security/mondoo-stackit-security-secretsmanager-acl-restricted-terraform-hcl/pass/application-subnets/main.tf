resource "stackit_secretsmanager_instance" "app" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name       = "app"
  acls       = ["10.0.0.0/8", "192.168.0.0/16"]
}
