// Copyright 2018 karma.run AG. All rights reserved.
package config

import (
	"flag"
	"log"
	"os"
)

var (
	HttpBind         string
	GethRPCURL       string
	CombinedJSONPath string
	FSAuthDirectory  string
)

var (
	LogWriter = os.Stderr
	LogFlags  = (log.Ldate | log.Ltime | log.Lshortfile)
)

func init() {
	flag.StringVar(
		&HttpBind,
		`http-bind`,
		getenv("KARMA_HTTP_PORT", ":8080"),
		`HTTP interface and port number to bind and serve`,
	)
	flag.StringVar(
		&GethRPCURL,
		`geth-rpc`,
		getenv("KARMA_GETH_RPC", ""),
		`URL or path to a running geth RPC API (local IPC pipe, WebSocket or HTTP)`,
	)
	flag.StringVar(
		&CombinedJSONPath,
		`combined-json`,
		getenv("KARMA_COMBINED_JSON", ""),
		`Path to combined.json file produced with solc --combined-json 'ast,bin'`,
	)
	flag.StringVar(
		&FSAuthDirectory,
		`fs-auth-dir`,
		getenv("KARMA_FS_AUTH_DIR", ""),
		`Path to auth/fs's private key directory`,
	)
}

func getenv(key, deflt string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return deflt
}
