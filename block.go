package main

import (
	"net/http"
	"strings"
)

func agentBlock(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		for _, bua := range []string{
			"Opera",
			"Thinkbot",
			"meta-webindexer",
			"node",
			"Uptime",
			"Amethyst",
			"babbar.tech",
			"semrush",
			"Bytespider",
			"AhrefsBot",
			"DataForSeoBot",
			"Yandex",
			"meta-externalagent",
			"DotBot",
			"ClaudeBot",
			"GPTBot",
			"MJ12Bot",
			"PetalBot",
			"Trident",
			"BLEXBot",
			"Aliyun",
			"Amazon",
			"Read-Aloud",
		} {
			if strings.Contains(ua, bua) {
				// log.Debug().Str("ua", ua).Msg("user-agent blocked")
				http.Error(w, "", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
