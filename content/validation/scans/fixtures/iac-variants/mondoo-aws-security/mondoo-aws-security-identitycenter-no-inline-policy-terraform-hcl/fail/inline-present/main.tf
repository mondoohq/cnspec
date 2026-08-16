# Non-compliant: an inline policy is attached to the permission set.
resource "aws_ssoadmin_permission_set_inline_policy" "fail_example" {
  instance_arn       = "arn:aws:sso:::instance/ssoins-example"
  permission_set_arn = "arn:aws:sso:::permissionSet/ssoins-example/ps-example"
  inline_policy      = "{}"
}
