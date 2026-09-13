# The VPC has its own active flow log capturing all traffic. The extra
# vSwitch flow log that records only rejected traffic adds detail and does not
# weaken the VPC's coverage.
resource "alicloud_vpc" "prod" {
  vpc_name   = "prod"
  cidr_block = "10.0.0.0/16"
}

resource "alicloud_vpc_flow_log" "prod" {
  flow_log_name  = "prod-vpc-flow"
  resource_id    = alicloud_vpc.prod.id
  resource_type  = "VPC"
  traffic_type   = "All"
  project_name   = "vpc-flow-logs"
  log_store_name = "prod-vpc"
  status         = "Active"
}

resource "alicloud_vpc_flow_log" "dmz_rejected" {
  flow_log_name  = "dmz-rejected-flow"
  resource_id    = alicloud_vswitch.dmz.id
  resource_type  = "VSwitch"
  traffic_type   = "Drop"
  project_name   = "vpc-flow-logs"
  log_store_name = "dmz-rejected"
  status         = "Active"
}
