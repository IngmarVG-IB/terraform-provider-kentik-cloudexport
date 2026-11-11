terraform {
  required_providers {
    kentik-cloudexport = {
      source = "kentik/kentik-cloudexport"
    }
  }
}

# Credentials are usually supplied through KTAPI_AUTH_EMAIL and KTAPI_AUTH_TOKEN.
provider "kentik-cloudexport" {
  # email  = "john@acme.com"
  # token  = "..."
  region = "EU" # US (default) or EU; or set endpoint = "host:port"
}
