# Non-compliant: no aws_ebs_snapshot_block_public_access resource is declared,
# so public sharing of EBS snapshots is not blocked account-wide.
resource "aws_ebs_snapshot" "fail_example" {
  volume_id = "vol-12345678"
}
