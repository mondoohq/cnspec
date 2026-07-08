# Non-compliant: a container pulls its image from Docker Hub, not OCIR.
resource "oci_container_instances_container_instance" "example" {
  compartment_id      = var.compartment_id
  availability_domain = var.availability_domain
  shape               = "CI.Standard.E4.Flex"

  shape_config {
    ocpus         = 1
    memory_in_gbs = 4
  }

  containers {
    display_name = "api"
    image_url    = "iad.ocir.io/mytenancy/api:1.4.2"
  }

  containers {
    display_name = "cache"
    image_url    = "docker.io/library/redis:7"
  }

  vnics {
    subnet_id = var.subnet_id
  }
}
