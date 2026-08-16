# Non-compliant: block-new-sharing only blocks new public sharing; snapshots
# already shared publicly remain restorable. Only block-all-sharing is compliant.
resource "aws_ebs_snapshot" "fail_example" {
  volume_id = "vol-12345678"
}

resource "aws_ebs_snapshot_block_public_access" "fail_example" {
  state = "block-new-sharing"
}
