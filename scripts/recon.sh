#!/bin/bash
RESOLVERS="/home/movsx/lists/resolvers.txt"
OUTPUT="/home/movsx/recon"
PROBE_PORTS="xlarge"
PRIOR_DATA=false
TARGET=""
TMPDIR="/tmp"
TMPEXT=".recon-$RANDOM"

usage() {
        echo -e "Usage: $0 [-h|-o <output directory>] domain\n"
        echo -e "-h\t\t\t\tthis help page"
        echo -e "-o <output directory>\t\toutput directory for data"

        exit
}

check_requirements() {
        tools=(jq curl findomain host massdns httprobe ffuf masscan)

        for i in ${tools[*]}
        do
                if [ ! `which "$i"` ]; then
                        echo "[!] Failed to find "$i". Exiting."
                        exit
                fi
        done
}

make_words() {
        echo -n "[*] Generating wordlist from subdomains.."
        awk -F. '{for (i=NF; i>1; --i) print $i}' $1 | tr '[:upper:]' '[:lower:]' | sort -u > "$OUTPUT/$1.words"
}

web_discover() {
	echo a
}

port_discover() {
	echo "port_discover(): $PROBE_PORTS, $TARGET, $OUTPUT"
        #httprobe -p "$PROBE_PORTS" "$TARGET" > "$OUTPUT.probe"
}

scrape() {
	echo "scrape(): $TARGET, $OUTPUT"
        findomain -q -t $TARGET -u "$OUTPUT.hosts" >/dev/null 2>/dev/null
        #file_len "$OUTPUT.hosts"
}

resolve() {
	echo "resolve(): $RESOLVERS, $OUTPUT"
        massdns -t A -o J -r "$RESOLVERS" -w "$OUTPUT.massdns" -q "$OUTPUT.hosts"
}

file_len() {
        if [ ! -f $1 ]; then
                echo 0
        fi

        wc -l $1 | awk '{print $1}'
}

check_prior() {
        for i in massdns probe hosts words valid
        do
                if [ -f "$OUTPUT.$i" ]; then
                	PRIOR_DATA=true
                        mv "$OUTPUT.$i" "$OUTPUT.$i.old"
                fi
        done
}

diff_results() {
        cur="$OUTPUT.massdns"
        if [[ -f "$cur" && -f "$cur.old" ]]; then
		jq '.query_name' "$cur" | sort -u > "$TMPDIR/valid.$TMPEXT"
                mv "$TMPDIR/valid.$TMPEXT" "$OUTPUT.valid"
                comm "$OUTPUT.valid" "$OUTPUT.valid.old" -3 | awk '{print $1}'
        fi
}

cleanup_handler() {
	rm -f "$TMPDIR/*.$TMPEXT"
	kill -9 $(pgrep --parent $$) 2>/dev/null
	echo; echo "[!] Cleaning up.."
	exit
}

main() {
        while getopts "ho:" OPTION; do
                case "${OPTION}" in
                        h)
                                usage
                                ;;
                        o)
                                OUTPUT=${OPTARG}
                                ;;
                        *)
                                usage
                                ;;
                esac
        done

        shift "$((OPTIND-1))"
        TARGET=$1

        if [[ -z "$TARGET" || -z "$OUTPUT" ]]; then
                usage
        fi
        OUTPUT="$OUTPUT/$TARGET"

        check_requirements

        trap cleanup_handler SIGINT
        echo "[*] Starting discovery of "$TARGET".."

        check_prior
	scrape
        resolve
        port_discover
        web_discover

        if [ "$PRIOR_DATA" == true ]; then
                echo "[*] New results:"
                diff_results
	else
		echo "[*] Results:"
		jq '.query_name' "$OUTPUT.massdns" | sort -u > "$OUTPUT.valid"
		wc -l "$OUTPUT.valid" | awk '{print $1}'
	fi

}

main $@
