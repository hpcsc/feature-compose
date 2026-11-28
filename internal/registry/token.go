package registry

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultUsername = "x-access-token"

type gitToken struct {
	server   *url.URL
	username string
	token    string
}

func tokensOfEnvironment() ([]gitToken, error) {
	var tokens []gitToken
	if token := cmp.Or(os.Getenv("GITHUB_TOKEN"), os.Getenv("GH_TOKEN")); token != "" {
		t, err := newGitToken("GITHUB_SERVER_URL", cmp.Or(os.Getenv("GITHUB_SERVER_URL"), "https://github.com"), defaultUsername, token)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	if token := os.Getenv("FEATURE_COMPOSE_GIT_TOKEN"); token != "" {
		server := os.Getenv("FEATURE_COMPOSE_GIT_SERVER")
		if server == "" {
			return nil, fmt.Errorf("FEATURE_COMPOSE_GIT_TOKEN is set, but FEATURE_COMPOSE_GIT_SERVER names no server: set it to the server of the token, for example https://gitlab.com")
		}
		t, err := newGitToken("FEATURE_COMPOSE_GIT_SERVER", server, cmp.Or(os.Getenv("FEATURE_COMPOSE_GIT_USERNAME"), defaultUsername), token)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func newGitToken(variable, server, username, token string) (gitToken, error) {
	parsed, err := url.Parse(server)
	if err != nil || parsed.Host == "" {
		return gitToken{}, fmt.Errorf("%s is %q, which is not the URL of a server, for example https://gitlab.com", variable, server)
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return gitToken{}, fmt.Errorf("%s is %s, but a token goes only to an https server, so that nobody on the network can read it", variable, server)
	}
	return gitToken{server: parsed, username: username, token: token}, nil
}

func (t gitToken) matches(source string) bool {
	parsed, err := url.Parse(source)
	return err == nil && parsed.Scheme == t.server.Scheme && strings.EqualFold(parsed.Host, t.server.Host)
}

func (t gitToken) environment(env []string) []string {
	// the environment, and not -c, holds the header, so that other users cannot read the token in the process list
	count, err := strconv.Atoi(os.Getenv("GIT_CONFIG_COUNT"))
	if err != nil {
		count = 0
	}
	header := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(t.username+":"+t.token))
	return append(env,
		"GIT_CONFIG_COUNT="+strconv.Itoa(count+1),
		fmt.Sprintf("GIT_CONFIG_KEY_%d=http.%s://%s/.extraheader", count, t.server.Scheme, t.server.Host),
		fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", count, header),
	)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
