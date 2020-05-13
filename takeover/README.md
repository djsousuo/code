# takeover
Subdomain takeover tool supporting concurrency, NXDOMAIN based takeovers, etc.
Sends positive results to Discord webhook

# Configuration
Edit config.json and fill out relevant details.

* discord
username/webhook to send discord events to
timeout is the delay between sending results

* general
nameserver: NS to use for lookups
user agent: self explanatory
concurrency: number of goroutines to use
retries: max # of retries to make for DNS queries

# Example Usage
`./takeover <file>`

`findomain -q -t site.com | ./takeover`

# TODO
* Add support for regex fingerprints
* Add support for multiple resolvers
