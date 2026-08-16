# Non-compliant: recorder has no recording_group block, so coverage of all
# supported and global resource types is not asserted.
resource "aws_config_configuration_recorder" "fail_no_group" {
  name     = "fail-no-group"
  role_arn = "arn:aws:iam::123456789012:role/config"
}
