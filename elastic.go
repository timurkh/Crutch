package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v5"
)

type ElasticHelper struct {
	client *elasticsearch.Client
}

type SearchQuery struct {
	Page        int    `json:"page"`
	Text        string `json:"text"`
	Category    string `json:"category"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Property    string `json:"property"`
	CityID      int    `json:"cityId"`
	InStockOnly bool   `json:"inStock"`
	Supplier    string `json:"supplier"`
}

func initElasticHelper(addr string) (*ElasticHelper, error) {

	cfg := elasticsearch.Config{
		Addresses: []string{
			addr,
		},
	}

	client, err := elasticsearch.NewClient(cfg)

	if err != nil {
		return nil, err
	}

	log.Println(elasticsearch.Version)
	log.Debug(client.Info())

	es := ElasticHelper{client}

	return &es, nil
}

func escapeSpecialSymbols(s string) string {
	return strings.Replace(
		strings.Replace(s, "/", "\\/", -1),
		":", "\\:", -1)

}

func escapeWildcardSymbols(s string) string {
	return strings.Replace(escapeSpecialSymbols(s), "*", "\\*", -1)
}

func (es *ElasticHelper) search(query *SearchQuery, ctx context.Context) (hits []interface{}, totalPages int, err error) {

	mustRequirementAnd := map[string]interface{}{
		"simple_query_string": map[string]interface{}{
			"query":            escapeWildcardSymbols(query.Text),
			"default_operator": "AND",
			"analyzer":         "russian_min_length_2",
			"fields": []interface{}{
				"code^3",
				"category^5",
				"name^2",
				"properties",
				"description",
			},
		},
	}

	mustRequirementOr := map[string]interface{}{
		"simple_query_string": map[string]interface{}{
			"query":            escapeWildcardSymbols(query.Text),
			"default_operator": "OR",
			"analyzer":         "russian_min_length_2",
			"fields": []interface{}{
				"code^3",
				"category^5",
				"name^2",
				"properties",
				"description",
			},
			"minimum_should_match": "50%",
		},
	}

	filterRequirements := make([]interface{}, 0)
	if len(query.Category) > 2 {
		filterRequirements = append(filterRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"category.name": query.Category,
			},
		},
		)
	}

	if len(query.Code) > 2 {
		filterRequirements = append(filterRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"code": query.Code,
			},
		},
		)
	}

	if len(query.Name) > 2 {
		filterRequirements = append(filterRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"name": query.Name,
			},
		},
		)
	}

	if len(query.Property) > 2 {
		filterRequirements = append(filterRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"properties.value": query.Property,
			},
		},
		)
	}

	var buf bytes.Buffer
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"bool": map[string]interface{}{
							"must":   mustRequirementAnd,
							"filter": filterRequirements,
						},
					},
					{
						"query_string": map[string]interface{}{
							"query": escapeSpecialSymbols(query.Text),
						},
					},
					{
						"bool": map[string]interface{}{
							"must":   mustRequirementOr,
							"filter": filterRequirements,
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
		"size": strconv.Itoa(itemsPerPage),
		"from": strconv.Itoa(query.Page * itemsPerPage),
	}

	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		return nil, 0, fmt.Errorf("Error encoding query: %v", err)
	}

	log.Debug("Quering elastic: ", buf.String())

	res, err := es.client.Search(
		es.client.Search.WithContext(ctx),
		es.client.Search.WithIndex("severstal_product"),
		es.client.Search.WithDocumentType("_doc"),
		es.client.Search.WithBody(&buf),
	)
	if err != nil {
		err = fmt.Errorf("Error getting response: %v", err)
		return nil, 0, err
	}

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, 0, fmt.Errorf("Error parsing the response body: %v", err)
		} else {
			// Print the response status and error information.
			return nil, 0, fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var response map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		err = fmt.Errorf("Error parsing elastic response: %v", err)
		return nil, 0, err
	}

	total := int(response["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	//temp workaround
	if total > 10000 {
		total = 10000
	}

	totalPages = total / itemsPerPage
	if totalPages*itemsPerPage < total {
		totalPages++
	}
	log.Debug("Status: ", res.Status(),
		", hits: ", total,
		", pages: ", totalPages,
		", items per page:", itemsPerPage,
		", iTook (ms): ", int(response["took"].(float64)),
	)

	hits = response["hits"].(map[string]interface{})["hits"].([]interface{})

	return hits, totalPages, nil
}
