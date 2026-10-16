package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var defaultTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SOA", "CAA", "DMARC"}

var extraTypes = []string{"HTTPS", "DNSKEY", "DS", "MTA-STS", "BIMI"}

type query struct {
	label string
	name  string
	typ   uint16
}

type record struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Value string `json:"value"`
}

type result struct {
	Type    string   `json:"type"`
	Records []record `json:"records"`
	Error   string   `json:"error,omitempty"`
}

func buildQueries(name string, types []string) ([]query, error) {
	name = dns.Fqdn(name)
	seen := map[string]bool{}
	var queries []query

	for _, t := range types {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true

		queryName := name
		rrtype := dns.TypeTXT
		switch t {
		case "DMARC":
			queryName = "_dmarc." + name
		case "MTA-STS":
			queryName = "_mta-sts." + name
		case "BIMI":
			queryName = "default._bimi." + name
		default:
			var ok bool
			rrtype, ok = dns.StringToType[t]
			if !ok {
				return nil, fmt.Errorf("unknown record type %q", t)
			}
		}

		queries = append(queries, query{label: t, name: queryName, typ: rrtype})
	}
	return queries, nil
}

func systemServer() string {
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || len(conf.Servers) == 0 {
		return "1.1.1.1:53"
	}
	return net.JoinHostPort(conf.Servers[0], conf.Port)
}

func serverAddr(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(strings.Trim(s, "[]"), "53")
}

type resolver struct {
	server  string
	timeout time.Duration
}

func (r *resolver) lookup(q query) result {
	res := result{Type: q.label, Records: []record{}}

	msg := new(dns.Msg)
	msg.SetQuestion(q.name, q.typ)
	msg.SetEdns0(4096, false)

	client := &dns.Client{Net: "udp", Timeout: r.timeout}
	resp, _, err := client.Exchange(msg, r.server)
	if err == nil && resp.Truncated {
		client.Net = "tcp"
		resp, _, err = client.Exchange(msg, r.server)
	}
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		res.Error = dns.RcodeToString[resp.Rcode]
		return res
	}

	for _, rr := range resp.Answer {
		h := rr.Header()
		if h.Rrtype != q.typ {
			continue
		}
		res.Records = append(res.Records, record{
			Name:  h.Name,
			Type:  dns.TypeToString[h.Rrtype],
			TTL:   h.Ttl,
			Value: rdata(rr),
		})
	}
	return res
}

func rdata(rr dns.RR) string {
	switch record := rr.(type) {
	case *dns.A:
		return record.A.String()
	case *dns.AAAA:
		return record.AAAA.String()
	case *dns.MX:
		return fmt.Sprintf("%d %s", record.Preference, record.Mx)
	case *dns.TXT:
		return strings.Join(record.Txt, "")
	default:
		return rr.String()
	}
}
