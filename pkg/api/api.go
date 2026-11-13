package api

import "net/url"

type PushTargetURL url.URL

func (p *PushTargetURL) String() string {
	return (*url.URL)(p).String()
}
