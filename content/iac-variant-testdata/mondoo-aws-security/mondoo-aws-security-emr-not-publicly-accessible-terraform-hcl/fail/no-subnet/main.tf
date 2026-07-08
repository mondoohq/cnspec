# Non-compliant: ec2_attributes block has no subnet_id, so the cluster is publicly accessible.
resource "aws_emr_cluster" "fail_example" {
  name          = "fail-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"

  ec2_attributes {
    instance_profile                  = "EMR_EC2_DefaultRole"
    emr_managed_master_security_group = "sg-master"
    emr_managed_slave_security_group  = "sg-slave"
  }
}
