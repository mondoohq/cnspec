# Compliant: account blocks all public sharing of EBS snapshots.
resource "aws_ebs_snapshot" "pass_example" {
  volume_id = "vol-12345678"
}

resource "aws_ebs_snapshot_block_public_access" "pass_example" {
  state = "block-all-sharing"
}
