# Compliant: cluster is deployed into a private subnet via ec2_attributes.
resource "aws_emr_cluster" "pass_example" {
  name          = "pass-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"

  ec2_attributes {
    subnet_id                         = "subnet-0123456789abcdef0"
    instance_profile                  = "EMR_EC2_DefaultRole"
    emr_managed_master_security_group = "sg-master"
    emr_managed_slave_security_group  = "sg-slave"
  }
}
