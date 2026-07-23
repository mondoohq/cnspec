# Non-compliant: DRS replication does not encrypt EBS volumes.
resource "aws_drs_replication_configuration_template" "fail_example" {
  associate_default_security_group = false
  bandwidth_throttling             = 0
  create_public_ip                 = false
  data_plane_routing               = "PRIVATE_IP"
  default_large_staging_disk_type  = "GP2"
  ebs_encryption                   = "NONE"
  replication_server_instance_type = "t3.small"
  use_dedicated_replication_server = false
}
