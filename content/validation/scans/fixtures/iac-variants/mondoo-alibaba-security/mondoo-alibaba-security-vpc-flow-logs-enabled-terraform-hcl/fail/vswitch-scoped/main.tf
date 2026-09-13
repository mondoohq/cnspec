# The only flow log watches one vSwitch, so traffic through the VPC's other
# vSwitches is never recorded and the VPC has no flow log of its own.
resource "alicloud_vpc_flow_log" "app" {
  flow_log_name  = "app-vswitch-flow"
  resource_id    = alicloud_vswitch.app.id
  resource_type  = "VSwitch"
  traffic_type   = "All"
  project_name   = "vpc-flow-logs"
  log_store_name = "app-vswitch"
  status         = "Active"
}
