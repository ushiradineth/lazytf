variable "message" {
  type = string
}

resource "null_resource" "child" {
  triggers = {
    message = var.message
  }
}

output "child_id" {
  value = null_resource.child.id
}
