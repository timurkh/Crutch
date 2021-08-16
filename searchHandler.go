package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/elastic/go-elasticsearch/v5"
)

const itemsPerPage = 5

type SearchHelper struct {
	es *elasticsearch.Client
}

type SearchQuery struct {
	Page     int    `json:"page"`
	Text     string `json:"text"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Property string `json:"property"`
}

func initSearchHelper() *SearchHelper {

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://10.130.0.21:9400",
		},
	}

	es, _ := elasticsearch.NewClient(cfg)

	log.Println(elasticsearch.Version)
	log.Println(es.Info())

	sh := SearchHelper{es}

	return &sh
}

func (sh *SearchHelper) searchProductsHandler(w http.ResponseWriter, r *http.Request) error {
	var searchQuery SearchQuery
	err := json.NewDecoder(r.Body).Decode(&searchQuery)
	if err != nil {
		return err
	}

	log.Printf("## Handling search request text=%s, category=%s, code=%s, name=%s, property=%s\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property)

	mustRequirements := make([]interface{}, 0)

	if len(searchQuery.Text) > 2 {
		mustRequirements = append(mustRequirements, map[string]interface{}{
			"simple_query_string": map[string]interface{}{
				"query":            searchQuery.Text,
				"default_operator": "AND",
				"analyzer":         "russian",
				"fields": []interface{}{
					"code^8",
					"description^2",
					"properties^2",
					"name^4",
					"category",
				},
			},
		})
	}

	if len(searchQuery.Category) > 2 {
		mustRequirements = append(mustRequirements, map[string]interface{}{
			"wildcard": map[string]interface{}{
				"category.name": searchQuery.Category + "*",
			},
		},
		)
	}

	if len(searchQuery.Code) > 2 {
		mustRequirements = append(mustRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"code": "*" + searchQuery.Code + "*",
			},
		},
		)
	}

	if len(searchQuery.Name) > 2 {
		mustRequirements = append(mustRequirements, map[string]interface{}{
			"wildcard": map[string]interface{}{
				"name": searchQuery.Name + "*",
			},
		},
		)
	}

	if len(searchQuery.Property) > 2 {
		mustRequirements = append(mustRequirements, map[string]interface{}{
			"match": map[string]interface{}{
				"properties.value": searchQuery.Property,
			},
		},
		)
	}

	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustRequirements,
			},
		},
		"size": strconv.Itoa(itemsPerPage),
		"from": strconv.Itoa(searchQuery.Page * itemsPerPage),
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
		err = fmt.Errorf("Error getting response: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
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
		err = fmt.Errorf("Error parsing elastic response: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	total := int(response["hits"].(map[string]interface{})["total"].(float64))
	totalPages := total / itemsPerPage
	if totalPages*itemsPerPage < total {
		totalPages++
	}
	// Print the response status, number of results, and request duration.
	log.Printf(
		"[%s] %d hits; took: %dms",
		res.Status(),
		total,
		int(response["took"].(float64)),
	)

	entries := sh.getResponseEntries(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Page       int         `json:"page"`
		TotalPages int         `json:"totalPages"`
		Results    interface{} `json:"results"`
	}{searchQuery.Page, totalPages, entries})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (sh *SearchHelper) getResponseEntries(response map[string]interface{}) []map[string]string {

	hits := response["hits"].(map[string]interface{})["hits"].([]interface{})

	entries := make([]map[string]string, len(hits))

	// iterate through hits
	for i, hit := range hits {
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
			"id":         h["_id"].(string),
			"categories": categories,
			"code":       s["code"].(string),
			"name":       s["name"].(string),
			"properties": properties,
		}

		entries[i] = entry
	}

	return entries

}
