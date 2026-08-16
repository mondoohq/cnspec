resource "gitlab_deploy_key" "example" {
  project  = gitlab_project.example.id
  title    = "example-deploy-key"
  key      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAExample"
  can_push = false
}
