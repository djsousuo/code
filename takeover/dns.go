package main

import (
	"github.com/miekg/dns"
	"strings"
	"fmt"
)

func dnsCheckTimeout(err error, tries int) bool {
	if strings.HasSuffix(err.Error(), "i/o timeout") && tries > 0 {
		return true
	}

	return false
}

func dnsCNAME(host string, tries int) (string, error) {
	var cname string

	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, Config.NS + ":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			return dnsCNAME(host, tries - 1)
		}

		return "", err
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.CNAME); ok {
			cname = t.Target
		}
	}

	return cname, nil
}

func dnsNS(host string, tries int) ([]string, error) {
	var ns []string

	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeNS)
	reply, err := dns.Exchange(msg, Config.NS + ":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			return dnsNS(host, tries - 1)
		}
		return nil, err
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.NS); ok {
			ns = append(ns, t.Ns)
		}
	}

	return ns, nil
}

func dnsA(host string, ns string, tries int) ([]string, error) {
	var a []string

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: false,
		},
	}
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, ns + ":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			return dnsA(host, ns, tries - 1)
		}
		return nil, err
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.A); ok {
			a = append(a, t.A.String())
		}
	}

	return a, nil
}
