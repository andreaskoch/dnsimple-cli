// Copyright 2016 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/pearkes/dnsimple"
	"net"
	"testing"
)

// testDNSCreator creates DNS records.
type testDNSCreator struct {
	createSubdomainFunc func(domain, subdomain string, timeToLive int, ip net.IP) error
}

func (creator *testDNSCreator) CreateSubdomain(domain, subdomain string, timeToLive int, ip net.IP) error {
	return creator.createSubdomainFunc(domain, subdomain, timeToLive, ip)
}

// If any of the given parameters is invalid CreateSubdomain should respond with an error.
func Test_CreateSubdomain_ParametersInvalid_ErrorIsReturned(t *testing.T) {
	// arrange
	inputs := []struct {
		domain    string
		subdomain string
		ttl       int
		ip        net.IP
	}{
		{"example.com", "", 600, net.ParseIP("::1")},
		{"www", "", 600, net.ParseIP("::1")},
		{"", "", 600, net.ParseIP("::1")},
		{" ", " ", 600, net.ParseIP("::1")},
		{"example.com", "www", 600, nil},
	}
	creator := dnsimpleCreator{}

	for _, input := range inputs {

		// act
		err := creator.CreateSubdomain(input.domain, input.subdomain, input.ttl, input.ip)

		// assert
		if err == nil {
			t.Fail()
			t.Logf("CreateSubdomain(%q, %q, %q, %q) should return an error.", input.domain, input.subdomain, input.ttl, input.ip)
		}
	}
}

// CreateSubdomain should return an error if the given subdomain does not exist.
func Test_CreateSubdomain_ValidParameters_SubdomainNotFound_ErrorIsReturned(t *testing.T) {
	// arrange
	domain := "example.com"
	subdomain := "www"
	ttl := 600
	ip := net.ParseIP("::1")

	infoProvider := &testDNSInfoProvider{
		getSubdomainRecordFunc: func(domain, subdomain, recordType string) (record dnsimple.Record, err error) {
			return dnsimple.Record{}, fmt.Errorf("")
		},
	}

	infoProviderFactory := testInfoProviderFactory{infoProvider}

	creator := dnsimpleCreator{
		infoProviderFactory: infoProviderFactory,
	}

	// act
	err := creator.CreateSubdomain(domain, subdomain, ttl, ip)

	// assert
	if err == nil {
		t.Fail()
		t.Logf("CreateSubdomain(%q, %q, %q, %q) should return an error if the subdomain does not exist.", domain, subdomain, ttl, ip)
	}
}

func Test_CreateSubdomain_ValidParameters_SubdomainExists_DNSRecordUpdateFails_ErrorIsReturned(t *testing.T) {
	// arrange
	domain := "example.com"
	subdomain := "www"
	ttl := 600
	ip := net.ParseIP("::1")

	dnsClient := &testDNSClient{
		createRecordFunc: func(domain string, opts *dnsimple.ChangeRecord) (string, error) {
			return "", fmt.Errorf("Record update failed")
		},
	}

	infoProvider := &testDNSInfoProvider{
		getSubdomainRecordFunc: func(domain, subdomain, recordType string) (record dnsimple.Record, err error) {
			return dnsimple.Record{}, nil
		},
	}

	dnsClientFactory := testDNSClientFactory{dnsClient}
	infoProviderFactory := testInfoProviderFactory{infoProvider}

	creator := dnsimpleCreator{
		clientFactory:       dnsClientFactory,
		infoProviderFactory: infoProviderFactory,
	}

	// act
	err := creator.CreateSubdomain(domain, subdomain, ttl, ip)

	// assert
	if err == nil {
		t.Fail()
		t.Logf("CreateSubdomain(%q, %q, %q) should return an error of the record update failed at the DNS client.", domain, subdomain, ip)
	}
}

func Test_CreateSubdomain_ValidParameters_SubdomainExists_DNSRecordUpdateSucceeds_NoErrorIsReturned(t *testing.T) {
	// arrange
	domain := "example.com"
	subdomain := "www"
	ttl := 3600
	ip := net.ParseIP("::1")

	dnsClient := &testDNSClient{
		createRecordFunc: func(domain string, opts *dnsimple.ChangeRecord) (string, error) {
			return "", nil
		},
	}

	infoProvider := &testDNSInfoProvider{
		getSubdomainRecordFunc: func(domain, subdomain, recordType string) (record dnsimple.Record, err error) {
			return dnsimple.Record{}, nil
		},
	}

	dnsClientFactory := testDNSClientFactory{dnsClient}
	infoProviderFactory := testInfoProviderFactory{infoProvider}

	creator := dnsimpleCreator{
		clientFactory:       dnsClientFactory,
		infoProviderFactory: infoProviderFactory,
	}

	// act
	err := creator.CreateSubdomain(domain, subdomain, ttl, ip)

	// assert
	if err != nil {
		t.Fail()
		t.Logf("CreateSubdomain(%q, %q, %q) should not return an error if the DNS record update succeeds.", domain, subdomain, ip)
	}
}

// If the update will not change the IP the update is aborted and an error is returned.
func Test_CreateSubdomain_ValidParameters_SubdomainExists_ExistingIPIsTheSame_ErrorIsReturned(t *testing.T) {
	// arrange
	domain := "example.com"
	subdomain := "www"
	ttl := 3600
	ip := net.ParseIP("::1")

	dnsClient := &testDNSClient{
		createRecordFunc: func(domain string, opts *dnsimple.ChangeRecord) (string, error) {
			return "", nil
		},
	}

	existingRecord := dnsimple.Record{
		Id:         1,
		Name:       "example.com",
		Content:    "::1",
		RecordType: "AAAA",
		Ttl:        600,
	}

	infoProvider := &testDNSInfoProvider{
		getSubdomainRecordFunc: func(domain, subdomain, recordType string) (record dnsimple.Record, err error) {
			return existingRecord, nil
		},
	}

	dnsClientFactory := testDNSClientFactory{dnsClient}
	infoProviderFactory := testInfoProviderFactory{infoProvider}

	creator := dnsimpleCreator{
		clientFactory:       dnsClientFactory,
		infoProviderFactory: infoProviderFactory,
	}

	// act
	err := creator.CreateSubdomain(domain, subdomain, ttl, ip)

	// assert
	if err == nil {
		t.Fail()
		t.Logf("CreateSubdomain(%q, %q, %q) should return an error because the IP of the existing record is the same as in the update.", domain, subdomain, ip)
	}
}

func Test_CreateSubdomain_ValidParameters_SubdomainExists_OnlyTheIPIsChangedOnTheDNSRecord(t *testing.T) {
	// arrange
	domain := "example.com"
	subdomain := "www"
	ttl := 3600
	ip := net.ParseIP("::2")

	existingRecord := dnsimple.Record{
		Id:         1,
		Name:       "example.com",
		Content:    "::1",
		RecordType: "AAAA",
		Ttl:        600,
	}

	dnsClient := &testDNSClient{
		createRecordFunc: func(domain string, opts *dnsimple.ChangeRecord) (string, error) {

			// assert
			if opts.Name != existingRecord.Name {
				t.Fail()
				t.Logf("The DNS name should not change during an update (Old: %q, New: %q)", existingRecord.Name, opts.Name)
			}

			if opts.Type != existingRecord.RecordType {
				t.Fail()
				t.Logf("The DNS record type should not change during an update (Old: %q, New: %q)", existingRecord.RecordType, opts.Type)
			}

			if opts.Ttl != fmt.Sprintf("%d", existingRecord.Ttl) {
				t.Fail()
				t.Logf("The DNS record TTL should not change during an update (Old: %q, New: %q)", existingRecord.Ttl, opts.Ttl)
			}

			if opts.Value != ip.String() {
				t.Fail()
				t.Logf("The DNS record value should have changed to %q", ip.String())
			}

			return "", nil
		},
	}

	infoProvider := &testDNSInfoProvider{
		getSubdomainRecordFunc: func(domain, subdomain, recordType string) (record dnsimple.Record, err error) {
			return existingRecord, nil
		},
	}

	dnsClientFactory := testDNSClientFactory{dnsClient}
	infoProviderFactory := testInfoProviderFactory{infoProvider}

	creator := dnsimpleCreator{
		clientFactory:       dnsClientFactory,
		infoProviderFactory: infoProviderFactory,
	}

	// act
	creator.CreateSubdomain(domain, subdomain, ttl, ip)
}
