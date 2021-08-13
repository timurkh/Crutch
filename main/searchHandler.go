package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/elastic/go-elasticsearch/v5"
)

type SearchHelper struct {
	es *elasticsearch.Client
}

type SearchQuery struct {
	Text     string `json:"text"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Property string `json:"property"`
}

func initSearchHelper() *SearchHelper {

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://10.130.0.21:9200",
		},
	}

	es, _ := elasticsearch.NewClient(cfg)

	log.Println(elasticsearch.Version)
	log.Println(es.Info())

	sh := SearchHelper{es}

	return &sh
}

func (sh *SearchHelper) searchProductsHandler(w http.ResponseWriter, r *http.Request) {
	var searchQuery SearchQuery
	err := json.NewDecoder(r.Body).Decode(&searchQuery)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("## Handling search request text=%s, category=%s, code=%s, name=%s, property=%s\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property)

	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"simple_query_string": map[string]interface{}{
				"query":    searchQuery.Text,
				"analyzer": "simple",
				"fields": []interface{}{
					"code^6",
					"description",
					"properties^2",
					"supplier_code^8",
					"name^4",
				},
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}

	res, err := sh.es.Search(
		sh.es.Search.WithContext(r.Context()),
		sh.es.Search.WithIndex("severstal"),
		sh.es.Search.WithDocumentType("model-Product"),
		sh.es.Search.WithBody(&buf),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			log.Fatalf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var response map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	// Print the response status, number of results, and request duration.
	log.Printf(
		"[%s] %d hits; took: %dms",
		res.Status(),
		int(response["hits"].(map[string]interface{})["total"].(float64)),
		int(response["took"].(float64)),
	)

	entries := sh.getResponseEntries(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(entries)
	if err != nil {
		log.Println(err.Error())
	}
}

func (sh *SearchHelper) getResponseEntries(response map[string]interface{}) []map[string]string {

	hits := response["hits"].(map[string]interface{})["hits"]

	entries := make([]map[string]string, len(hits))

	// iterate through hits
	for i, hit := range hits.([]interface{}) {
		h := hit.(map[string]interface{})
		s := h["_source"].(map[string]interface{})
		var categories string

		//categories
		for i, c := range s["category"].([]interface{}) {
			if i > 0 {
				categories += ", "
			}
			cat := c.(map[string]interface{})["name"]
			if cat != nil {
				categories += cat.(string)
			}
		}

		//properties
		var properties string
		if s["properties"] != nil {
			for i, p := range s["properties"].([]interface{}) {
				if i > 0 {
					properties += ", "
				}
				properties += p.(map[string]interface{})["property"].(map[string]interface{})["name"].(string)
				properties += " : "
				properties += p.(map[string]interface{})["value"].(string)
			}
		}

		entry := map[string]string{
			"categories": categories,
			"code":       s["code"].(string),
			"name":       s["name"].(string),
			"properties": properties,
		}

		entries[i] = entry

		log.Printf("\n%d: ID=%s, Category=%s, Code=%s, Name=%s, Properties=%s\n", i, h["_id"], categories, s["code"], s["name"], properties)

	}

	return entries

}
