# Compliant: DRS template omits create_public_ip, so recovery instances get no public IP.
resource "aws_drs_replication_configuration_template" "pass_example" {
  associate_default_security_group = false
  bandwidth_throttling             = 0
  data_plane_routing               = "PRIVATE_IP"
  default_large_staging_disk_type  = "GP2"
  ebs_encryption                   = "DEFAULT"
  replication_server_instance_type = "t3.small"
  use_dedicated_replication_server = false
}
