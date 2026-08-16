resource "aws_snapshot_create_volume_permission" "noncompliant" {
  snapshot_id = "snap-1234567890abcdef0"
  account_id  = "123456789012"
}
