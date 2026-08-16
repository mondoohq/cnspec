resource "openstack_identity_user_v3" "service" {
  name               = "svc-backup"
  default_project_id = openstack_identity_project_v3.ops.id
  description        = "Backup service account"

  ignore_lockout_failure_attempts = true
}
