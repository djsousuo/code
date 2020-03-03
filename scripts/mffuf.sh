#!/bin/bash
xargs -P10 -I % ffuf -u %/FUZZ -w wordlist.txt -o output-%.json < domains.txt
