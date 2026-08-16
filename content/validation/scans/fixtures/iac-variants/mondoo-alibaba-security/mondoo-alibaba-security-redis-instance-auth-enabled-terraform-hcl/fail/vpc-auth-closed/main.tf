# "Close" disables password authentication for VPC clients, so anything that
# can reach the instance can read and write every key.
resource "alicloud_kvstore_instance" "cache" {
  db_instance_name = "prod-redis"
  instance_class   = "redis.master.small.default"
  instance_type    = "Redis"
  engine_version   = "7.0"
  vswitch_id       = alicloud_vswitch.cache.id
  vpc_auth_mode    = "Close"
}
