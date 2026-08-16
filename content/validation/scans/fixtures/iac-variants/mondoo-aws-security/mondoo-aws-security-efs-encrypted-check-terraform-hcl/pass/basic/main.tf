resource "aws_efs_file_system" "pass" {
  creation_token = "pass"
  encrypted      = true
  kms_key_id     = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
}
