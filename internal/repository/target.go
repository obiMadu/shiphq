package repository

import (
	"fmt"
	"os/exec"
	"strings"
)

type Host string

const (
	HostUnknown   Host = "unknown"
	HostGitHub    Host = "github"
	HostGitLab    Host = "gitlab"
	HostBitbucket Host = "bitbucket"
)

type Target struct {
	Host      Host
	RemoteURL string
}

func DetectTarget() (Target, error) {
	command := exec.Command("git", "remote", "get-url", "origin")
	output, err := command.CombinedOutput()
	if err != nil {
		return Target{}, fmt.Errorf("failed to detect repository target: %w\n%s", err, output)
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return Target{}, fmt.Errorf("git remote origin URL is empty")
	}

	return ParseTarget(remoteURL), nil
}

func ParseTarget(remoteURL string) Target {
	lowercaseRemoteURL := strings.ToLower(remoteURL)
	host := HostUnknown

	switch {
	case strings.Contains(lowercaseRemoteURL, "github.com"):
		host = HostGitHub
	case strings.Contains(lowercaseRemoteURL, "gitlab"):
		host = HostGitLab
	case strings.Contains(lowercaseRemoteURL, "bitbucket"):
		host = HostBitbucket
	}

	return Target{
		Host:      host,
		RemoteURL: remoteURL,
	}
}
