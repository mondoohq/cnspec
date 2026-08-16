resource "openstack_db_instance_v1" "reporting" {
  name      = "reporting-postgres"
  flavor_id = "10"
  size      = 8

  datastore {
    type    = "postgresql"
    version = "13"
  }

  network {
    uuid = "3c8e2f1a-4b5c-4d6e-8f90-1a2b3c4d5e6f"
  }
}
