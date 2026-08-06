package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const dbaRootUrl = "https://www.dba.dk/recommerce/forsale/search/api/search/SEARCH_ID_BAP_COMMON"

type FetcherService struct {
	client *http.Client
}

func NewFetcherService() FetcherService {
	return FetcherService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func getDbaParameterizedUrl(query string) (string, error) {
	reqUrl, err := url.Parse(dbaRootUrl)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("dealer_segment", "1")
	params.Add("sort", "PUBLISHED_DESC")
	params.Add("q", query)
	params.Add("page", "1")

	reqUrl.RawQuery = params.Encode()
	return reqUrl.String(), nil
}

func (fetcherSvc *FetcherService) FetchDba(ctx context.Context, query string) ([]DbaDoc, error) {
	reqUrl, err := getDbaParameterizedUrl(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := fetcherSvc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from dba api: %d", resp.StatusCode)
	}

	var dbaResults DbaSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&dbaResults); err != nil {
		return nil, fmt.Errorf("failed to decode dba response: %w", err)
	}
	return dbaResults.Docs, nil
}

