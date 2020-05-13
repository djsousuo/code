package main

import (
	"github.com/miekg/dns"
	"strings"
)

func dnsCheckTimeout(err error, tries int) bool {
	if strings.HasSuffix(err.Error(), "i/o timeout") && tries > 0 {
		return true
	}

	return false
}

func dnsCNAME(host string) (string, error) {
	var cname string

	tries := Config.Retries
	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, Config.NS+":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			tries--
			return dnsCNAME(host)
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

func dnsNS(host string) ([]string, error) {
	var ns []string

	tries := Config.Retries
	msg := new(dns.Msg)
	msg.SetQuestion(host, dns.TypeNS)
	reply, err := dns.Exchange(msg, Config.NS+":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			tries--
			return dnsNS(host)
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

func dnsA(host string, ns string) ([]string, error) {
	var a []string

	tries := Config.Retries
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: false,
		},
	}
	msg.SetQuestion(host, dns.TypeA)
	reply, err := dns.Exchange(msg, ns+":53")
	if err != nil {
		if dnsCheckTimeout(err, tries) {
			tries--
			return dnsA(host, ns)
		}
		return nil, err
	}

	for _, answer := range reply.Answer {
		if t, ok := answer.(*dns.A); ok {
			a = append(a, t.A.String())
		}
	}

	return a, nil
	/*
		if reply.Rcode == dns.RcodeNameError {
			return ns, true
		}
		return "", false

		if reply.Rcode == dns.RcodeServerFailure || reply.Rcode == dns.RcodeRefused {
			return ns, true
		}
		return "", false
	*/
}

/*
func dnsNX(host string) bool {
	if _, err := net.LookupHost(host); err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return true
		}
	}
	return false
}
*/
