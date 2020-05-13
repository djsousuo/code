# azure
Check availability of domain takeovers on Azure

Requires the following environment variables to be setup and exported
* AZURE_TENANT_ID
* AZURE_CLIENT_ID
* AZURE_CLIENT_SECRET

The "subscription" variable must also be updated to match your settings

See https://community.microfocus.com/t5/Identity-Manager-Tips/Creating-the-application-Client-ID-and-Client-Secret-from/ta-p/1776619 for more details.

# Example Usage
`./azure name[.trafficmanager.net]`
