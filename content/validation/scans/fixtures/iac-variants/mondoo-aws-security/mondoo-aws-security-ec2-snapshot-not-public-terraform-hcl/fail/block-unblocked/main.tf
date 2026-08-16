resource "aws_ebs_snapshot_block_public_access" "this" {
  state = "unblocked"
}
