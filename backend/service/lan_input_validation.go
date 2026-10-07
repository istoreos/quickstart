package service

import (
	"errors"
	"net"
	"net/netip"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	dhcpHostnamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	dhcpTagNamePattern  = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]{0,63}$`)
)

func normalizeStaticAssignmentInput(input StaticAssignmentWriteInput) (StaticAssignmentWriteInput, error) {
	input.Action = strings.TrimSpace(input.Action)
	input.AssignedMAC = strings.ToUpper(strings.TrimSpace(input.AssignedMAC))
	input.AssignedIP = strings.TrimSpace(input.AssignedIP)
	input.Hostname = strings.ToLower(strings.TrimSpace(input.Hostname))
	input.TagName = strings.TrimSpace(input.TagName)
	input.TagTitle = strings.TrimSpace(input.TagTitle)

	hardwareAddr, err := net.ParseMAC(input.AssignedMAC)
	if err != nil || len(hardwareAddr) != 6 {
		return StaticAssignmentWriteInput{}, errors.New("a valid MAC address is required")
	}
	input.AssignedMAC = strings.ToUpper(hardwareAddr.String())
	if input.Action == "delete" {
		input.AssignedIP = ""
		input.BindIP = false
		input.Hostname = ""
		input.TagName = ""
		input.TagTitle = ""
		input.MaterializeAutoTag = false
		return input, nil
	}

	if input.Hostname != "" && !dhcpHostnamePattern.MatchString(input.Hostname) {
		return StaticAssignmentWriteInput{}, errors.New("DHCP hostname must be a 1-63 character ASCII label using only letters, numbers, and interior hyphens")
	}
	if input.AssignedIP != "" {
		address, parseErr := netip.ParseAddr(input.AssignedIP)
		if parseErr != nil || !address.Is4() {
			return StaticAssignmentWriteInput{}, errors.New("a valid IPv4 address is required")
		}
		input.AssignedIP = address.String()
	}
	if input.BindIP && input.Action != "delete" && input.AssignedIP == "" {
		return StaticAssignmentWriteInput{}, errors.New("an IPv4 address is required when MAC binding is enabled")
	}
	if input.TagName != "" && !dhcpTagNamePattern.MatchString(input.TagName) {
		return StaticAssignmentWriteInput{}, errors.New("invalid DHCP route identifier")
	}
	if !validSafeLabel(input.TagTitle, 128) {
		return StaticAssignmentWriteInput{}, errors.New("invalid DHCP route title")
	}
	return input, nil
}

func validSafeLabel(value string, maxBytes int) bool {
	if !utf8.ValidString(value) || len(value) > maxBytes {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f || char == '\'' {
			return false
		}
	}
	return true
}
