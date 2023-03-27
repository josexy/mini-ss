#!/bin/bash

# configure local DNS server manually
function updateDNS {
    case "$1" in
    g | google)
        echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
        ;;
    a | ali)
        echo "nameserver 223.5.5.5" | sudo tee /etc/resolv.conf
        ;;
    l | local)
        echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
        ;;
    esac
}

updateDNS l
