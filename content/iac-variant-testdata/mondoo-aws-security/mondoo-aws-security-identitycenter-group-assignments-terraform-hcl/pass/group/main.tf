# Compliant: assignment principal is a GROUP.
resource "aws_ssoadmin_account_assignment" "pass_example" {
  instance_arn       = "arn:aws:sso:::instance/ssoins-example"
  permission_set_arn = "arn:aws:sso:::permissionSet/ssoins-example/ps-example"
  principal_id       = "example-group-id"
  principal_type     = "GROUP"
  target_id          = "123456789012"
  target_type        = "AWS_ACCOUNT"
}
