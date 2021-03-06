// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudflare

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/op/go-logging"

	"github.com/pulcy/quark/providers"
)

const (
	apiUrl = "https://api.cloudflare.com/client/v4/"
)

type cfProvider struct {
	Logger *logging.Logger
	apiKey string
	email  string
}

func NewProvider(logger *logging.Logger, apiKey, email string) providers.DnsProvider {
	return &cfProvider{
		Logger: logger,
		apiKey: apiKey,
		email:  email,
	}
}

type cfResponse struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Success bool            `json:"success"`
}

func (r *cfResponse) UnmarshalResult(v interface{}) error {
	if err := json.Unmarshal(r.Result, v); err != nil {
		return maskAny(err)
	}
	return nil
}

func (p *cfProvider) request(method, url string, payload io.Reader, headers map[string]string) (*cfResponse, error) {
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, maskAny(err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Auth-Key", p.apiKey)
	req.Header.Set("X-Auth-Email", p.email)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, maskAny(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, maskAny(err)
	}

	var resp cfResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, maskAny(err)
	}

	return &resp, nil
}

func (p *cfProvider) get(url, contentType string) (*cfResponse, error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}

	res, err := p.request("GET", url, nil, headers)
	if err != nil {
		return nil, maskAny(err)
	}

	return res, nil
}

func (p *cfProvider) delete(url string) (*cfResponse, error) {
	res, err := p.request("DELETE", url, nil, nil)
	if err != nil {
		return nil, maskAny(err)
	}

	return res, nil
}

func (p *cfProvider) post(url, contentType string, payload io.Reader) (*cfResponse, error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}

	res, err := p.request("POST", url, payload, headers)
	if err != nil {
		return nil, maskAny(err)
	}

	return res, nil
}

func (p *cfProvider) postJson(url string, body interface{}) (*cfResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, maskAny(err)
	}

	return p.post(url, "application/json", bytes.NewReader(data))
}
