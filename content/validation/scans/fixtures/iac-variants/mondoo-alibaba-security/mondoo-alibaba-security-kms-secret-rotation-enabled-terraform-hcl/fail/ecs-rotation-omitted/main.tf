# The generic secret is out of scope, but the ECS secret beside it supports
# automatic rotation and leaves it at the default of off.
resource "alicloud_kms_secret" "api_token" {
  secret_name = "api-token"
  secret_type = "Generic"
  secret_data = var.api_token
  version_id  = "v1"
}

resource "alicloud_kms_secret" "ecs_login" {
  secret_name     = "acs/ecs/web-01"
  secret_type     = "ECS"
  secret_data     = jsonencode({ UserName = "root", Password = var.ecs_password })
  extended_config = jsonencode({ CommandId = "c-example", InstanceId = "i-example" })
  version_id      = "v1"
}
