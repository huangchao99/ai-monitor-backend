package zlm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"ai-monitor-backend/config"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func get(path string, params url.Values) (json.RawMessage, error) {
	params.Set("secret", config.ZLMSecret)
	u := config.ZLMBaseURL + path + "?" + params.Encode()
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r apiResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("zlm response parse error: %w, body=%s", err, body)
	}
	if r.Code != 0 {
		return nil, fmt.Errorf("zlm error code=%d msg=%s", r.Code, r.Msg)
	}
	return r.Data, nil
}

// AddStreamProxy asks ZLM to pull an RTSP stream and re-publish it.
// Returns the proxy key assigned by ZLM.
func AddStreamProxy(streamKey, rtspURL string) (string, error) {
	params := url.Values{
		"vhost":       {"__defaultVhost__"},
		"app":         {config.ZLMApp},
		"stream":      {streamKey},
		"url":         {rtspURL},
		"enable_rtsp": {"1"},
		"enable_rtmp": {"1"},
		"enable_hls":  {"1"},
		"rtp_type":    {"0"},
	}
	data, err := get("/index/api/addStreamProxy", params)
	if err != nil {
		return "", err
	}
	var d struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return "", err
	}
	return d.Key, nil
}

// DelStreamProxy removes a ZLM proxy stream by key.
func DelStreamProxy(proxyKey string) error {
	params := url.Values{"key": {proxyKey}}
	_, err := get("/index/api/delStreamProxy", params)
	return err
}

// IsStreamAlive returns true if ZLM has an active media stream for the given app/stream.
func IsStreamAlive(streamKey string) bool {
	params := url.Values{
		"vhost":  {"__defaultVhost__"},
		"app":    {config.ZLMApp},
		"stream": {streamKey},
	}
	data, err := get("/index/api/getMediaInfo", params)
	if err != nil {
		return false
	}
	var d struct {
		AliveSecond int `json:"aliveSecond"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return false
	}
	return d.AliveSecond >= 0
}

// ServerStatus returns true if ZLM is reachable.
func ServerStatus() bool {
	params := url.Values{}
	_, err := get("/index/api/version", params)
	return err == nil
}

// BuildFlvURL returns the HTTP-FLV URL for a stream.
func BuildFlvURL(streamKey string) string {
	return fmt.Sprintf("%s/%s/%s.live.flv", config.ZLMBaseURL, config.ZLMApp, streamKey)
}

// BuildHlsURL returns the HLS URL for a stream.
func BuildHlsURL(streamKey string) string {
	return fmt.Sprintf("%s/%s/%s/hls.m3u8", config.ZLMBaseURL, config.ZLMApp, streamKey)
}
