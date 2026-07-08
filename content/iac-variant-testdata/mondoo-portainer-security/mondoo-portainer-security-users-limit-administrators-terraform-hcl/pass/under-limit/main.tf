resource "portainer_user" "admin" {
  username = "admin"
  role     = 1
}

resource "portainer_user" "platform_lead" {
  username = "platform-lead"
  role     = 1
}

resource "portainer_user" "operator" {
  username = "operator"
  role     = 2
}

resource "portainer_user" "developer" {
  username = "developer"
  role     = 2
}
