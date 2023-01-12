#!/bin/bash

function scutil_query {
    key=$1
    scutil <<EOT
open
get $key
d.show
close
EOT
}

function updateDNS {
    SERVICE_GUID=$(scutil_query State:/Network/Global/IPv4 | grep "PrimaryService" | awk '{print $3}')
    currentservice=$(scutil_query Setup:/Network/Service/$SERVICE_GUID | grep "UserDefinedName" | awk -F': ' '{print $2}')
    echo "Current active networkservice is \"$currentservice\", $SERVICE_GUID"
    olddns=$(networksetup -getdnsservers "$currentservice")

    case "$1" in
    d | default)
        echo "old dns is $olddns, set dns to default"
        networksetup -setdnsservers "$currentservice" empty
        ;;
    g | google)
        echo "old dns is $olddns, set dns to google dns"
        networksetup -setdnsservers "$currentservice" 8.8.8.8 4.4.4.4
        ;;
    a | ali)
        echo "old dns is $olddns, set dns to alidns"
        networksetup -setdnsservers "$currentservice" "223.5.5.5"
        ;;
    1 | 114)
        echo "old dns is $olddns, set dns to 114dns"
        networksetup -setdnsservers "$currentservice" "114.114.114.114"
        ;;
    l | local)
        echo "old dns is $olddns, set dns to 127.0.0.1"
        networksetup -setdnsservers "$currentservice" "127.0.0.1"
        ;;
    c | custom)
        echo "old dns is $olddns, set dns to custom $2"
        networksetup -setdnsservers "$currentservice" "$2"
        ;;
    esac
}

function flushCache {
    echo "flush cache"
    sudo dscacheutil -flushcache
    sudo killall -HUP mDNSResponder
}

updateDNS $1 $2
flushCache
