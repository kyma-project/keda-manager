# BTP integration test

## Overview

This integration test verifies if Keda Manager works in a semi-production environment.

## How to use

Export the following environment variables:
```bash
TF_VAR_BTP_BOT_USER=
TF_VAR_BTP_BOT_PASSWORD=
TF_VAR_BTP_GLOBAL_ACCOUNT=
TF_VAR_BTP_CUSTOM_IAS_TENANT=
```

You can use the following command to export variables from the `.env` file which contains the above variables:
```bash
export $(cat .env | xargs)
```

