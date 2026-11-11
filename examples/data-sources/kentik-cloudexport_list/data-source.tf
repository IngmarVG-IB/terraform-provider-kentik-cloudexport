# List all cloud exports in the account.
data "kentik-cloudexport_list" "exports" {}

output "export_names" {
  value = [for e in data.kentik-cloudexport_list.exports.items : e.name]
}

output "aws_exports" {
  value = [for e in data.kentik-cloudexport_list.exports.items : e.id if e.cloud_provider == "aws"]
}
