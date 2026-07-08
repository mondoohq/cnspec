# Non-compliant: assignment principal is a USER instead of a GROUP.
resource "aws_ssoadmin_account_assignment" "fail_example" {
  instance_arn       = "arn:aws:sso:::instance/ssoins-example"
  permission_set_arn = "arn:aws:sso:::permissionSet/ssoins-example/ps-example"
  principal_id       = "example-user-id"
  principal_type     = "USER"
  target_id          = "123456789012"
  target_type        = "AWS_ACCOUNT"
}
