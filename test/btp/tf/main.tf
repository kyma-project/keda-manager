terraform {
  required_providers {
    btp = {
      source  = "SAP/btp"
      version = "1.7.0"
    }
    jq = {
      source  = "massdriver-cloud/jq"
    }
    http = {
      source = "hashicorp/http"
      version = "3.4.5"
    }
  }
}

provider "jq" {}
provider "http" {}

provider "btp" {
  globalaccount = var.BTP_GLOBAL_ACCOUNT
  cli_server_url = var.BTP_BACKEND_URL
  idp            = var.BTP_CUSTOM_IAS_TENANT
  username = var.BTP_BOT_USER
  password = var.BTP_BOT_PASSWORD
}

module "kyma" {
  source = "git::https://github.com/kyma-project/terraform-module.git?ref=v0.2.0"
  BTP_NEW_SUBACCOUNT_NAME = var.BTP_NEW_SUBACCOUNT_NAME
  BTP_CUSTOM_IAS_TENANT = var.BTP_CUSTOM_IAS_TENANT
  BTP_BOT_USER = var.BTP_BOT_USER
  BTP_BOT_PASSWORD = var.BTP_BOT_PASSWORD
}

data "btp_subaccount_service_binding" "provider_sm" {
  count = var.BTP_PROVIDER_SUBACCOUNT_ID == null ? 0 : 1
  subaccount_id = var.BTP_PROVIDER_SUBACCOUNT_ID
  name          = "provider-sm-binding"
}

locals {
  providerServiceManagerCredentials = var.BTP_PROVIDER_SUBACCOUNT_ID == null ? null : jsondecode(one(data.btp_subaccount_service_binding.provider_sm).credentials)
}


resource "local_file" "provider_sm" {
  count = var.BTP_PROVIDER_SUBACCOUNT_ID == null ? 0 : 1
  content  = <<EOT
clientid=${local.providerServiceManagerCredentials.clientid}
clientsecret=${local.providerServiceManagerCredentials.clientsecret}
sm_url=${local.providerServiceManagerCredentials.sm_url}
tokenurl=${local.providerServiceManagerCredentials.url}
tokenurlsuffix=/oauth/token
EOT
  filename = "provider-sm-decoded.env"
}


output "subaccount_id" {
  value = module.kyma.subaccount_id
}
