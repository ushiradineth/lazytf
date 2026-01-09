terraform {
  required_version = ">= 1.3.0"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = ">= 3.2.2"
    }
  }
}

resource "null_resource" "example" {
  triggers = {
    message = "hello"
    count   = 1
  }
}

output "example_id" {
  value = null_resource.example.id
}
