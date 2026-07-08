resource "openstack_db_instance_v1" "cache" {
  name      = "app-redis"
  flavor_id = "10"
  size      = 8

  datastore {
    type    = "redis"
    version = "6.0"
  }

  network {
    uuid = "3c8e2f1a-4b5c-4d6e-8f90-1a2b3c4d5e6f"
  }
}
