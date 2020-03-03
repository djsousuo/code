#!/bin/sh
METHODS="^(GET|HEAD|POST|PUT|PATCH|DELETE|OPTIONS)"
strings *.burp > .tmp.burp
egrep "$METHODS" .tmp.burp | awk '{print $2}' | cut -d\? -f1 | sort -u | tee -a paths
egrep "$METHODS" .tmp.burp | awk ' BEGIN { FS="?" } { n=split($2,b,/&/); for(i=1;i<=n;i++) print b[i] }' > .tmp.params
egrep "^[[:alnum:]]+=[[:alnum:]]*\&[[:alnum:]]*=[[:alnum:]]*" .tmp.burp | awk '{n=split($1,b,/&/); for(i=1;i<=n;i++) print b[i]}' >> .tmp.params
cut -d= -f1 .tmp.params | sort -u | tee -a params
rm -f .tmp.params .tmp.burp
