# takeover
Subdomain takeover tool supporting concurrency, NXDOMAIN based takeovers, and more

# Configuration
Config has been ripped from subjack and modified with new and up to date entries

# Example Usage
`./takeover <file>`

`findomain -q -t site.com | ./takeover`

# TODO
* Add support for regex fingerprints
* Eliminate false positives (s3, fastly)
