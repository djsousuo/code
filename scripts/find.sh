#!/bin/bash
#
# domain discovery script that utilizes
# - crt.sh
# - cert spotter
# - findomain
# - sublist3r
#
# then runs discovery on wildcard subdomains that were found
#
SUBDOMAINS=0

usage() {
       	echo -e "Usage: $0 [-h|-r|-e|-b <list>] <domain> <output file>\n"
	echo -e "-h\t\tusage"
	echo -e "-r\t\trecursively scan wildcard subdomains that are found"
	echo -e "-e\t\textract all possible subdomains from results and attempt to discover"
	echo -e "-b <list>\tbrute force subdomains using <list>"

	exit
}

check_requirements() {
	tools=(jq curl findomain sublist3r host)

	for i in ${tools[*]}
	do
		if [ ! `which "$i"` ]; then
			echo "[-] Failed to find $i. Exiting."
			exit
		fi
	done
}

scrape() {
	resp=$(curl --fail -s "https://crt.sh/?output=json&q=%.$1")
	if [[ $resp =~ "Fails." ]]; then
        	print "[-] crt.sh failed."
	fi

	echo $resp | jq --args '.[] | .name_value' | awk -F '"' '{print $2}'

	resp=$(curl --fail -s "https://certspotter.com/api/v0/certs?domain=$1")
	if [[ $resp =~ "Fails." ]]; then
		print "[-] certspotter failed"
	fi

	echo $resp | jq --args '.[].dns_names[]' | cut -d\" -f2
}

run_discovery() {
	findomain -q -t $1 >> $2 2>/dev/null
	scrape $1 >> $2
}

on() {
	case "$1" in 1) echo "ON" ;; *) echo "OFF" ;; esac
}

main() {
	UUID=$RANDOM
	TMP=/tmp/.discover.$UUID

	while getopts "hreb:" OPTION; do
		case "${OPTION}" in
		h)
			usage
			;;
		r) 
			SUBDOMAINS=1
			;;
		b)
			BRUTE=1
			WORDLIST=${OPTARG}
			;;
		e)
			EXTRACT=1
			;;
		*)
			usage
			;;
		esac
	done


	shift "$((OPTIND-1))"
	TARGET=$1
	OUT=$2

	if [[ -z "$TARGET" || -z "$OUT" ]]; then
		usage
	fi

	if [ -f $OUT ]; then
		echo "[-] File already exists: $OUT"
		exit
	fi

	if [[ "$BRUTE" -eq 1 && ! -f $WORDLIST ]]; then
		echo "[-] Can't open wordlist: $WORDLIST"
		exit
	fi

	check_requirements

	echo "[*] Discovering $TARGET.."
	echo "[*] Output: $OUT"
	echo -n "[*] Wildcard domain discovery: "
		on $SUBDOMAINS
	echo -n "[*] Sub domain wordlist discovery: "
		on $BRUTE
	echo -n "[*] Sub domain extraction: "
		on $EXTRACT


	trap "rm -f $TMP; kill -9 $(pgrep --parent $$) 2>/dev/null; echo;echo '[*] Cleaning up'; exit" SIGINT
	run_discovery $TARGET $TMP

	# pull out the "*.domain.com" wildcards and run those
	if [ "$SUBDOMAINS" -eq 1 ]; then
		if [ $(egrep -c "^\*." $TMP) -eq "0" ]; then
			echo " None found."
		else
			for i in $(egrep "^\*." $TMP | cut -b3- | sort -u); do
				echo -ne "\r[*] Discovering wildcard domains: $i                      \r"
				run_discovery $i $TMP
			done
			echo -e "\r[*] Discovering wildcard domains: Done                    \r"
		fi
	fi

	# basic brute force via wordlist
	# TODO: extract various names from parts of discovered domains and try those in variations
	if [ ! -z "$BRUTE" ]; then
		echo -n "[*] Brute forcing $(wc -l $WORDLIST|awk '{print $1}') domains from ($WORDLIST):"
		for i in $(cat $WORDLIST); do
			try="$i.$TARGET"

			# see if we get any dns records before running it
			host -t ANY "$try" >/dev/null
			if [ $? -eq 1 ]; then
				continue
			fi

			echo -ne "\r[+] Trying:  $try                                                \r"
			run_discovery $try $TMP
		done
		echo -e "[*] Brute forcing: Done                                                    \r"
	fi

	# extract subdomains from our results and try discovery on various permutations
	if [ ! -z "$EXTRACT" ]; then
		echo -n "[*] Extracting sub domains from results.. "
		awk -F. '{for (i=NF; i>1; --i) print $i}' $TMP | tr '[:upper:]' '[:lower:]' | sort -u > /tmp/.extract
		echo "$(wc -l /tmp/.extract|awk '{print $1}') found"
		for i in $(cat /tmp/.extract); do
			try="$i.$TARGET"
			host -t ANY "$try" >/dev/null
			if [ $? -eq 1 ]; then
				continue
			fi
			echo -ne "\r[+] Trying: $try                                              \r"
			run_discovery $try $TMP
		done
		echo -e "\r[*] Subdomain discovery: Done                                               \r"
	fi

	egrep -v "^\[|^\*" $TMP | sort -u >> $OUT
	echo -n "[+] Total results: "
	wc -l $OUT | awk '{print $1}'

	rm -f $TMP /tmp/.extract
}

main $@
