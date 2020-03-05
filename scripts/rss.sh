#!/bin/bash
# tool to scrape RSS feeds from torrent sites and send to qBitTorrent
#
FEED="https://rss.com/feed"
PROXY="192.168.1.1:8118"
URL="http://192.168.1.1:8080"
USER="user"
PASSWORD="password"
RSS_FILE=".rss.tmp"
IFS=$'\r\n'

echo -n "Grabbing RSS feed and logging into qBitTorrent..."
curl -s -x $PROXY -o $RSS_FILE $FEED
cookie=$(curl -s -i --header "Referer: $URL" --data "username=$USER&password=$PASSWORD" \
        $URL/login | awk -F 'SID=' '{print $2}' | cut -f1 -d\;)
echo "OK."
echo ""

if [ -f $RSS_FILE ]; then
        title=($(yq-xq '.rss.channel.item[] | .title' $RSS_FILE | cut -d'"' -f 2))
        link=($(yq-xq '.rss.channel.item[] | .link' $RSS_FILE | cut -d'"' -f 2))
        for i in ${!title[@]}; do
                torrent="${title[i]}.torrent"
                if [ ! -f $torrent ]; then
                        curl -s -x $PROXY -o $torrent ${link[i]}
                        curl -s -H "Referer: $URL/upload.html" \
                                -H "Cookie: SID=$cookie" \
                                -F "torrents=@$torrent" \
                                -F 'category=MaM' \
                                $URL/command/upload
                        echo "Adding $torrent"
                fi
        done
        rm $RSS_FILE
fi
