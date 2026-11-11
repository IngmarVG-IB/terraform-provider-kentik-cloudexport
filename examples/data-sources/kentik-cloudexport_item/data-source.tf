# Read a single cloud export by ID. The export must exist.
data "kentik-cloudexport_item" "export" {
  id = "3"
}

output "export_status" {
  value = data.kentik-cloudexport_item.export.current_status
}
