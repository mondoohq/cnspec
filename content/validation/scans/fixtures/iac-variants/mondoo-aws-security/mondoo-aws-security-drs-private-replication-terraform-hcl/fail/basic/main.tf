# Non-compliant: DRS replication routes over public IP.
resource "aws_drs_replication_configuration_template" "fail_example" {
  associate_default_security_group = false
  bandwidth_throttling             = 0
  create_public_ip                 = false
  data_plane_routing               = "PUBLIC_IP"
  default_large_staging_disk_type  = "GP2"
  ebs_encryption                   = "DEFAULT"
  replication_server_instance_type = "t3.small"
  use_dedicated_replication_server = false
}
