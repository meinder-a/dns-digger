# DNS Digger

The purpose of this tool is to have a small lightweight cli tool that exposes multiple dns records in one shot
I simply wanted to be able to have A, AAAA, CNAME, MX and TXT records all at once, thus, i made this tool

## Install

```sh
go install github.com/meinder-a/dns-digger@latest
```

## How to use it

```sh
dns-digger example.com
```

```
A      172.66.147.243
       104.20.23.154
AAAA   2606:4700:10::6814:179a
       2606:4700:10::ac42:93f3
MX     0 .
TXT    v=spf1 -all
       _k2n1y4vw3qtb4skdx9e7dxt97qrmmq9
NS     hera.ns.cloudflare.com.
       elliott.ns.cloudflare.com.
SOA    elliott.ns.cloudflare.com. dns.cloudflare.com. 2415555295 10000 2400 604800 1800
DMARC  v=DMARC1;p=reject;sp=reject;adkim=s;aspf=s
```

Specify the types you want by listing them after the domain

```sh
dns-digger google.fr A MX TXT
```

Give it an IP and it does a reverse lookup instead

```sh
dns-digger 1.1.1.1
```

### Flags

| Flag              | Description                                                                |
| ----------------- | -------------------------------------------------------------------------- |
| `-all`            | also look up HTTPS, DNSKEY, DS, MTA-STS and BIMI                           |
| `-server 1.1.1.1` | ask this server instead of the system resolver |
| `-json`           | print JSON                                                                 |
| `-timeout 5s`     | timeout                                                                    |
| `-version`        | show the version                                                           |

### Extra types

There are a few shortcuts for TXT records that live on well known subdomains

| Type      | Looks up                 |
| --------- | ------------------------ |
| `DMARC`   | `_dmarc.<domain>`        |
| `MTA-STS` | `_mta-sts.<domain>`      |
| `BIMI`    | `default._bimi.<domain>` |

### More examples

```sh
dns-digger _xmpp-server._tcp.jabber.org SRV
dns-digger _443._tcp.example.com TLSA
dns-digger -all -server 8.8.8.8 -json example.com
```

On systems without `/etc/resolv.conf`, like Windows, it uses `1.1.1.1` unless `-server` is specified

## License

MIT
