#!/bin/bash
OUT=$2

if [ ! $1 ]; then
        echo "Usage: $0 <domain.com> <out>"
        exit
fi

if [ ! -f `which jq` ]; then
        echo "Exiting. Install jq json processor"
        exit
fi

echo "Scraping crt.sh"
resp=$(curl --fail -s "https://crt.sh/?output=json&q=%.$1")

if [[ $resp =~ "Fails." ]]; then
        echo "crt.sh failed."
        exit
fi

echo $resp | jq --args '.[] | .name_value' | awk -F '"' '{print $2}' >> .crtsh.tmp.$1

echo "Scraping certspotter"
resp=$(curl --fail -s "http://certspotter.com/api/v0/certs?domain=$1")
if [[ $resp =~ "Fails." ]]; then
        echo "certspotter failed"
fi

echo $resp | jq --args '.[].dns_names[]' | cut -d\" -f2 >> .certspotter.tmp.$1

echo "Running findomain..."
findomain -t $1 -u .findomain.tmp.$1

echo "Combining and sorting results..."
cat .findomain.tmp.$1 .certspotter.tmp.$1 .crtsh.tmp.$1 | sort -u > $2
