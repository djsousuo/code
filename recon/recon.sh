#!/bin/bash
# BACKUP is the main archive directory that output will be stored in
BACKUP="/mnt/recon/data/archive"
TASKLOG="$BACKUP/task.log"
ALLSTO="$BACKUP/hosts.takeover"
ALLHOSTS="$BACKUP/hosts.all"
ALLPROBE="$BACKUP/hosts.probe"
ALLRESOLVED="$BACKUP/hosts.resolved"
# list of wildcard domains to run findomain on
WILDCARDS="/home/movsx/lists/wildcards.txt"
# massdns resolvers
RESOLVERS="/home/movsx/lists/r.txt"
# postgresql settings
PGHOST='localhost'
PGUSER='postgres'
export PGPASSWORD='yourpassword'
# bounty-targets-data github repo
BOUNTY_TARGETS="/home/movsx/repo/bounty-targets-data"
# directory and config file for findomain
FINDOMAIN_CONFIG="/home/movsx/findomain/config.json"
TAKEOVER_PATH="/home/movsx/takeover"
UPDATE_MODE=0
########################################################################

usage() {
	echo "Usage: $0 [-h|-u]"
	echo "-h\tusage"
	echo "-u\tupdate mode (only perform scans on new data found instead of whole dataset)"

	exit
}

tasklog() {
	echo "$1 at $(date +%c)" >> "$TASKLOG"
}

update_wildcards() {
	if [ -d "$BOUNTY_TARGETS" ]; then
		echo -n "Updating bounty-targets-data... "
		(cd "$BOUNTY_TARGETS" && git pull) >/dev/null 2>/dev/null
		echo "Done"
		NEW_WILDCARDS="$BOUNTY_TARGETS/data/wildcards.txt"
		if [[ -f "$WILDCARDS" && -f "$NEW_WILDCARDS" ]]; then
			cat "$NEW_WILDCARDS" | egrep "^\*\." | cut -c3- | egrep -v "sip*twilio.com" | sed -e 's/\.\*$/\.com/g' > .wild
			cat .wild "$WILDCARDS" | sort -u > .tmp
			mv .tmp "$WILDCARDS"
			rm -f .wild .tmp
		else
			echo -n "$WILDCARDS and $NEW_WILDCARDS not found. exiting"
			exit
		fi
	else
		echo -n "bounty-targets-data not found. exiting"
		exit
	fi
}

check_lastrun() {
	if [ "$(( $(date +"%s") - $(stat -c "%Y" "$1") ))" -gt "7200" ]; then
		return TRUE
	fi
}

compare_results() {
	NEW="$1"
	ORIGINAL="$NEW.last"

	if [ ! -f "$NEW" ]; then
		tasklog "$NEW not found. exiting"
		exit
	fi

	if [ -f "$ORIGINAL" ]; then
		ARCHIVE="$NEW-$(date +%d%m%y-%H:%M).tgz"
		tasklog "archiving historical data: $ORIGINAL -> $ARCHIVE"
		comm -3 "$NEW" "$ORIGINAL" | awk '{$1=$1};1' > "$NEW.diff"
		tar -zcf "$ARCHIVE" "$ORIGINAL" "$NEW.diff"
		DIFF=$(wc -l "$NEW.diff"|awk '{print $1}')
		tasklog "$DIFF new changes on $NEW"
	else
		tasklog "new run saved $NEW"
		cp "$NEW" "$ORIGINAL"
	fi
}

find_sto() {
	echo "[*] running takeover"
	tasklog "takeover scan starting"

	case "$UPDATE_MODE" in
		0) INPUT="$ALLHOSTS" ;;
		1) INPUT="$ALLHOSTS.diff" ;;
	esac

	cat "$INPUT" | (cd "$TAKEOVER_PATH"; ./takeover | tee /tmp/output.sto)
	sort -u /tmp/output.sto > "$ALLSTO"
	rm -f /tmp/output.sto
	compare_results "$ALLSTO"
	tasklog "takeover scan completed"
}

find_domains() {
	echo "[*] running findomain"
	tasklog "started findomain task"
	findomain -q -m -c "$FINDOMAIN_CONFIG" -f "$WILDCARDS"
	tasklog "findomain task ended"
}

resolve_domains() {
	if [[ ! -f "$ALLHOSTS" || ! -f "$RESOLVERS" ]]; then
		tasklog "error: $ALLHOSTS or $RESOLVERS doesn't exist. exiting"
		exit
	fi
	echo "[*] running massdns"
	tasklog "started massdns task"
	massdns -r "$RESOLVERS" -q "$ALLHOSTS" -o S -t A | awk '{print $1}' | sort -u | awk -F '\.$' '{print $1}' > "$ALLRESOLVED"
	tasklog "massdns task ended"
	compare_results "$ALLRESOLVED"
}

probe() {
	echo "[*] running httprobe"
	tasklog "started httprobe task"


	case "$UPDATE_MODE" in
		0) INPUT="$ALLRESOLVED" ;;
		1) INPUT="$ALLRESOLVED.diff" ;;
	esac

	cat "$INPUT" | httprobe -p xlarge > "$ALLPROBE"
	tasklog "httprobe task ended"
	compare_results "$ALLPROBE"
}

run_massget() {
	# placeholder
	tasklog "started massget (not really)"
}

psql_gethosts() {
	psql --username=postgres --host=localhost --quiet -c "SELECT name FROM subdomains_fdplus" -o /tmp/output.hosts
	cat /tmp/output.hosts | egrep ".*\..*" | cut -d" " -f2 | sort -u > "$ALLHOSTS"
	rm -f /tmp/output.hosts
	compare_results "$ALLHOSTS"
}

int_handler() {
        kill -9 $(pgrep --parent $$) 2>/dev/null
        echo; echo '[*] got ^C. cleaning up'
        tasklog '------------------ terminating processes'
        exit
}

main() {
	while getopts "hu" OPTION; do
		case "${OPTION}" in
			h)
				usage
				;;
			u)
				UPDATE_MODE=1
				;;
			*)
				usage
				;;
		esac
	done

	if [ "$UPDATE_MODE" == 1 ]; then
		echo "[*] running in update mode"
		all_files=($ALLHOSTS.last $ALLRESOLVED.last)
		for i in ${all_files[*]}; do
			if [ ! -f "$i" ]; then
				echo "[-] specified update mode without an initial run. exiting."
				exit
			fi
		done
	fi
	tasklog "-------------------- started new session"

	trap int_handler SIGINT
	# main functions here
	update_wildcards
	find_domains
	psql_gethosts
	resolve_domains
	# probe &
	find_sto
	tasklog "-------------------- session ended"
}

main $@
