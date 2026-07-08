resource "openstack_identity_user_v3" "admin" {
  name               = "platform-admin"
  default_project_id = openstack_identity_project_v3.ops.id
  description        = "Platform administrator"
}
