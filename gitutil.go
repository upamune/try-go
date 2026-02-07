package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	httpsGitRE = regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+?)(?:\.git)?$`)
	sshGitRE   = regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
)

type gitTarget struct {
	Host string
	User string
	Repo string
}

func parseGitURI(raw string) (gitTarget, bool) {
	raw = strings.TrimSpace(raw)
	if m := httpsGitRE.FindStringSubmatch(raw); m != nil {
		return gitTarget{Host: m[1], User: m[2], Repo: m[3]}, true
	}
	if m := sshGitRE.FindStringSubmatch(raw); m != nil {
		return gitTarget{Host: m[1], User: m[2], Repo: m[3]}, true
	}
	return gitTarget{}, false
}

func isGitURI(raw string) bool {
	_, ok := parseGitURI(raw)
	return ok
}

func cloneDirName(basePath, rawURL, custom string) (string, error) {
	if strings.TrimSpace(custom) != "" {
		name := sanitizeName(custom)
		if name == "" {
			return "", fmt.Errorf("invalid custom name")
		}
		today := time.Now().Format("2006-01-02")
		return uniqueDirName(basePath, fmt.Sprintf("%s-%s", today, name)), nil
	}
	gt, ok := parseGitURI(rawURL)
	if !ok {
		return "", fmt.Errorf("unable to parse git URI: %s", rawURL)
	}
	today := time.Now().Format("2006-01-02")
	return uniqueDirName(basePath, fmt.Sprintf("%s-%s-%s", today, gt.User, gt.Repo)), nil
}
