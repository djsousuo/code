# takeover
* Subdomain takeover tool supporting hosted as well as dangling CNAME based takeovers.
* Concurrent and supports multiple resolvers, as well as custom timeouts for DNS/HTTP
* Option to send positive results to Discord webhook
* 50+ fingerprints for vulnerable services

# Configuration
Edit config.json

## Discord Configuration
* username: Name that messages will be displayed from
* webhook: URL to send events to
* timeout: Delay between sending results
* maxentries: Number of results to send per request

## General Settings
* resolvers: Path to text file containing DNS resolvers
* user_agent: UA to send for HTTP requests
* concurrency: Number of goroutines to use
* timeout: Timeout for requests
* retries: Maximum # of retries to make for DNS queries that timeout

# Example Usage
`./takeover <file>`

`findomain -q -t site.com | ./takeover`

# TODO
* Add support for regex fingerprints
* Verify results for false positives (Azure services, Webflow, etc)
