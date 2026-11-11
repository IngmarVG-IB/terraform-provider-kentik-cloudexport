# For local development use a dev_overrides CLI config, see AGENTS.md.
terraform {
  required_providers {
    kentik-cloudexport = {
      version = ">= 0.3.0"
      source  = "kentik/kentik-cloudexport"
    }
  }
}

provider "kentik-cloudexport" {}
