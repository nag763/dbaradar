package fetcher

import (
	"encoding/json"
	"net/http"
	"net/url"
)

const dbaRootUrl = "https://www.dba.dk/recommerce/forsale/search/api/search/SEARCH_ID_BAP_COMMON"

type FetcherService struct {
}

func NewFetcherService() FetcherService {
	return FetcherService{}
}

func getDbaParameterizedUrl(query string) (string, error) {
	reqUrl, err := url.Parse(dbaRootUrl)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("dealer_segment", "1")
	params.Add("sort", "PUBLISHED_DESC")
	params.Add("Content-Type", "application/json")
	params.Add("Accept", "application/json")
	params.Add("q", query)
	params.Add("page", "1")

	reqUrl.RawQuery = params.Encode()
	return reqUrl.String(), nil
}

func (fetcherSvc *FetcherService) FetchDba(query string) ([]DbaDoc, error) {

	reqUrl, err := getDbaParameterizedUrl(query)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dbaResults DbaSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&dbaResults); err != nil {
		return nil, err
	}
	return dbaResults.Docs, nil

}
