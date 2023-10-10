# Reset Rabbit

## Intro

A hastily thrown together PoC for testing for DoS on http/2 webservers. It performs the technique described in https://www.cve.org/CVERecord?id=CVE-2023-44487

## Usage

First start by installing go on your machine.

Included is a Dockerfile for an apache webserver that is vulnerable to this type of attack. You can run it like so:

    docker build -t vulnerable-apache ./vulnerable-apache
    docker run -p 8443:443 vulnerable-apache

Then run it like so:

    go run reset-rabbit.go -url https://localhost:8443 -limit 1

In a few seconds, the web server will no longer be accessible and requests will just hang. You may have to adjust the limit to take down bigger servers with more resources.

## Notes

No idea if this works in real life, but in small test apps it seems to do the trick.

The app is kinda hard on your machine. You might need to `kill -9` the process to get it to exit.
