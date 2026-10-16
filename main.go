package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const version = "1.0.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type options struct {
	server      string
	allRecords  bool
	jsonOutput  bool
	timeout     time.Duration
	showVersion bool
	showHelp    bool
}

func run(args []string, stdout, stderr io.Writer) error {
	opts, positional, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}
	if opts.showHelp {
		return nil
	}

	if opts.showVersion {
		_, err := fmt.Fprintln(stdout, version)
		return err
	}
	if len(positional) == 0 {
		return fmt.Errorf("a domain or IP address is required")
	}

	target := positional[0]
	queries, err := queriesForTarget(target, positional[1:], opts.allRecords)
	if err != nil {
		return err
	}

	server := systemServer()
	if opts.server != "" {
		server = serverAddr(opts.server)
	}
	resolver := &resolver{server: server, timeout: opts.timeout}
	results := make([]result, 0, len(queries))
	for _, query := range queries {
		results = append(results, resolver.lookup(query))
	}

	if opts.jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report{Target: target, Server: server, Results: results})
	}
	printText(stdout, results)
	return nil
}

func parseFlags(args []string, stderr io.Writer) (options, []string, error) {
	var opts options
	flags := flag.NewFlagSet("dns-digger", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.server, "server", "", "DNS server to query; uses the system resolver by default")
	flags.BoolVar(&opts.allRecords, "all", false, "query every common record type")
	flags.BoolVar(&opts.jsonOutput, "json", false, "print results as JSON")
	flags.DurationVar(&opts.timeout, "timeout", 5*time.Second, "timeout for each query")
	flags.BoolVar(&opts.showVersion, "version", false, "print the version")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: dns-digger [flags] <domain or ip> [types...]")
		fmt.Fprintln(stderr, "example: dns-digger google.fr A MX TXT")
		fmt.Fprintln(stderr, "\nflags:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return options{showHelp: true}, nil, nil
		}
		return options{}, nil, err
	}
	return opts, flags.Args(), nil
}

func queriesForTarget(target string, types []string, allRecords bool) ([]query, error) {
	name := target
	selected := defaultTypes
	if allRecords {
		selected = append(append([]string{}, defaultTypes...), extraTypes...)
	}

	if net.ParseIP(target) != nil {
		var err error
		name, err = dns.ReverseAddr(target)
		if err != nil {
			return nil, err
		}
		selected = []string{"PTR"}
	}
	if len(types) > 0 {
		selected = []string{}
		for _, value := range types {
			selected = append(selected, strings.Split(value, ",")...)
		}
	}
	return buildQueries(name, selected)
}

type report struct {
	Target  string   `json:"target"`
	Server  string   `json:"server"`
	Results []result `json:"results"`
}

func printText(w io.Writer, results []result) {
	width := 0
	for _, res := range results {
		if len(res.Type) > width {
			width = len(res.Type)
		}
	}

	found := false
	for _, res := range results {
		if res.Error != "" {
			fmt.Fprintf(w, "%-*s  error: %s\n", width, res.Type, res.Error)
			found = true
			continue
		}
		for i, rec := range res.Records {
			label := res.Type
			if i > 0 {
				label = ""
			}
			fmt.Fprintf(w, "%-*s  %s\n", width, label, rec.Value)
			found = true
		}
	}
	if !found {
		fmt.Fprintln(w, "no records found")
	}
}
