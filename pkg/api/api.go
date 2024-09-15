package api

import "net/url"

type PushTargetUrl url.URL

func (p *PushTargetUrl) String() string {
	return (*url.URL)(p).String()
}
