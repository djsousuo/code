# parseburp
Tool for parsing BurpSuite sessions into JSON. Designed for piping to other tools such as csrfr.

The following fields are output:
* time
* url
* host
* port
* protocol
* method
* path
* request (request body)
* status
* responselength
* mimetype
* response (response body)
* headers (request headers)
* params (request parameters)

# Saving Session Data
Select entries to be saved in BurpSuite, right click and "Save" to file.

# Example Usage
Extract all javascript:

`./parseburp <session> | jq -j 'select (.status == 200 and .responselength > 0) | .time, " ", .url, " ", .protocol," ", .mimetype, "\n"' | grep SCRIPT`

Extract all unique entries where parameters include "url":

`./parseburp <session> | jq 'select(.params | contains("url")) | .url' | sort -u`

Generate csrfr data:

`./parseburp <session> | jq -s '[.[] | {url, method, params}]' | csrfr`
