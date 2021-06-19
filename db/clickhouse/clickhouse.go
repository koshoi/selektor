package clickhouse

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/koshoi/selektor/config"
	"github.com/koshoi/selektor/db/options"
)

type CHClient struct {
	Env config.EnvConfig
}

func (ch *CHClient) queryURL(query string) string {
	return fmt.Sprintf(
		"%s/?user=%s&password=%s&query=%s",
		ch.Env.Endpoint,
		ch.Env.User,
		url.QueryEscape(ch.Env.Password),
		url.QueryEscape(query),
	)
}

func (ch *CHClient) Query(_ context.Context, query string, opt options.Options) (io.Reader, error) {
	resp, err := http.Get(ch.queryURL(query))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		byteResp, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("non 2XX status=%d, failed to read body", resp.StatusCode)
		} else {
			return nil, fmt.Errorf("non 2XX status=%d: %s", resp.StatusCode, string(byteResp))
		}
	}

	return resp.Body, nil
}
