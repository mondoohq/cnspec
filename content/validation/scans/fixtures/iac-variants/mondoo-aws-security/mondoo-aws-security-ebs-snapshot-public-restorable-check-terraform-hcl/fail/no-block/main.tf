# Non-compliant: public sharing of EBS snapshots is not blocked.
resource "aws_ebs_snapshot" "fail_example" {
  volume_id = "vol-12345678"
}

resource "aws_ebs_snapshot_block_public_access" "fail_example" {
  state = "unblocked"
}
