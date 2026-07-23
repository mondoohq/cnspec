resource "openstack_db_instance_v1" "mysql" {
  name      = "app-mysql"
  flavor_id = "10"
  size      = 8

  datastore {
    type    = "mysql"
    version = "8.0"
  }

  network {
    uuid = "3c8e2f1a-4b5c-4d6e-8f90-1a2b3c4d5e6f"
  }
}

resource "openstack_db_instance_v1" "mariadb" {
  name      = "app-mariadb"
  flavor_id = "10"
  size      = 8

  datastore {
    type    = "mariadb"
    version = "10.11"
  }

  network {
    uuid = "3c8e2f1a-4b5c-4d6e-8f90-1a2b3c4d5e6f"
  }
}

resource "openstack_db_instance_v1" "postgres" {
  name      = "app-postgres"
  flavor_id = "10"
  size      = 8

  datastore {
    type    = "postgresql"
    version = "15"
  }

  network {
    uuid = "3c8e2f1a-4b5c-4d6e-8f90-1a2b3c4d5e6f"
  }
}
