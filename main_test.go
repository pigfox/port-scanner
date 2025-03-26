package main

import (
	"testing"
	"time"
)

func TestIpToUint32(t *testing.T) {
	tests := []struct {
		ip   string
		want uint32
	}{
		{"192.168.1.1", 3232235777},
		{"10.0.0.1", 167772161},
		{"255.255.255.255", 4294967295},
	}

	for _, tt := range tests {
		got := ipToUint32(tt.ip)
		if got != tt.want {
			t.Errorf("ipToUint32(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestUint32ToIP(t *testing.T) {
	tests := []struct {
		num  uint32
		want string
	}{
		{3232235777, "192.168.1.1"},
		{167772161, "10.0.0.1"},
		{4294967295, "255.255.255.255"},
	}

	for _, tt := range tests {
		got := uint32ToIP(tt.num)
		if got != tt.want {
			t.Errorf("uint32ToIP(%v) = %q, want %q", tt.num, got, tt.want)
		}
	}
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"80,443,8080", []int{80, 443, 8080}},
		{"22", []int{22}},
		{"80,invalid,443", []int{80, 443}},
		{"", []int{}},
		{"65536,0,-1", []int{}},
	}

	for _, tt := range tests {
		got := parsePorts(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parsePorts(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parsePorts(%q) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestScanPort(t *testing.T) {
	limiter := make(chan struct{}, 1)
	result := scanPort("127.0.0.1", 80, 100*time.Millisecond, limiter)

	if result.IP != "127.0.0.1" || result.Port != 80 {
		t.Errorf("scanPort returned wrong IP or port: got %s:%d", result.IP, result.Port)
	}
}
