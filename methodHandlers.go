package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/xuri/excelize/v2"
)

const itemsPerPage = 100

type MethodHandlers struct {
	auth *AuthMiddleware
	es   *ElasticHelper
	db   *DBHelper
}

func initMethodHandlers(auth *AuthMiddleware, es *ElasticHelper, db *DBHelper) *MethodHandlers {

	mh := MethodHandlers{auth, es, db}

	return &mh
}

func stripSpecialSymbols(s string) string {
	re := regexp.MustCompile(`[*:;\.\\/()]`)
	return re.ReplaceAllString(s, " ")
}

func (mh *MethodHandlers) searchProductsHandler(w http.ResponseWriter, r *http.Request) (err error) {

	userInfo := mh.auth.getUserInfo(r)

	var cities []City

	if !userInfo.Admin {
		cities, err = mh.db.getUserConsigneeCities(r.Context(), userInfo)
		if err != nil {
			err = fmt.Errorf("Failed to get consignee cities: %s", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}

		if len(cities) == 0 {
			err = fmt.Errorf("Current user does not have any warehouses assigned")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}

	var searchQuery SearchQuery
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(&searchQuery, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode search params: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	log.Printf("## Handling search request text=%s, category=%s, code=%s, name=%s, property=%s, page=%v\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property, searchQuery.Page)
	totalPages := 0
	entries := make([]map[string]interface{}, 0)
	for {
		hits, totalPages_, err := mh.es.search(&searchQuery, r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		entries_, err := mh.getResponseEntries(r.Context(), hits, userInfo, searchQuery.CityID, searchQuery.InStockOnly, searchQuery.Supplier)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		entries = append(entries, entries_...)
		totalPages = totalPages_

		if len(entries) > itemsPerPage/2 || searchQuery.Page >= totalPages-1 {
			break
		}
		searchQuery.Page++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities     []City      `json:"cities"`
		Page       int         `json:"page"`
		TotalPages int         `json:"totalPages"`
		Results    interface{} `json:"results"`
	}{userInfo, cities, searchQuery.Page, totalPages, entries})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getResponseEntries(ctx context.Context, hits []interface{}, userInfo UserInfo, cityId int, inStockOnly bool, supplier string) ([]map[string]interface{}, error) {

	ids := make([]int, len(hits))

	// iterate through hits
	products_score := make(map[int]float64, 0)
	for i, hit := range hits {
		h := hit.(map[string]interface{})
		id, _ := strconv.Atoi(h["_id"].(string))
		ids[i] = id
		score, _ := h["_score"].(float64)
		products_score[id] = score
	}

	log.Debug("Quering details for product_ids ", products_score)

	products, err := mh.db.getProductEntries(ctx, ids, products_score, userInfo, cityId, inStockOnly, supplier)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve list of products: %v", err)
	}

	return products, err
}

func (mh *MethodHandlers) getResponseEntriesFromElastic(hits []interface{}) []map[string]string {

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

func (mh *MethodHandlers) getCurrentUser(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	cities, err := mh.db.getUserConsigneeCities(r.Context(), userInfo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities []City `json:"cities"`
	}{userInfo, cities})

	log.Info(userInfo)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getCounterpartsHandler(w http.ResponseWriter, r *http.Request) error {

	var filter CounterpartsFilter
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&filter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userInfo := mh.auth.getUserInfo(r)

	if !userInfo.Admin {
		http.Error(w, "This resource requires admin privileges", http.StatusUnauthorized)
		return err
	}

	log.Info("Getting list of counterparts, filter ", filter)

	counterparts, err := mh.db.getCounterparts(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Counterparts interface{} `json:"counterparts"`
	}{counterparts})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrdersHandler(w http.ResponseWriter, r *http.Request) error {

	var ordersFilter OrdersFilter
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&ordersFilter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userInfo := mh.auth.getUserInfo(r)

	log.Info("Getting list of orders, filter ", ordersFilter)

	orders, err := mh.db.getOrders(r.Context(), userInfo, ordersFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Orders interface{} `json:"orders"`
	}{orders})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrderHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	params := mux.Vars(r)
	orderId_ := params["orderId"]
	orderId, err := strconv.Atoi(orderId_)
	if err != nil {
		err = fmt.Errorf("Failed to determin requested order ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	orderDetails, err := mh.db.getOrder(r.Context(), userInfo, orderId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(orderDetails)

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (mh *MethodHandlers) getOrdersExcelHandler(w http.ResponseWriter, r *http.Request) error {

	var ordersFilter OrdersFilter
	err := schema.NewDecoder().Decode(&ordersFilter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	file, err := os.CreateTemp("/tmp", "*.xlsx")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer os.Remove(file.Name())

	xls := excelize.NewFile()
	streamWriter, err := xls.NewStreamWriter("Sheet1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{Value: "[8]"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "(2)"},
		excelize.Cell{Value: "(2а)"},
		excelize.Cell{Value: "(2б)"},
		excelize.Cell{Value: "(3)"},
		excelize.Cell{},
		excelize.Cell{Value: "(4)"},
		excelize.Cell{},
		excelize.Cell{Value: "(6) и [19]"},
		excelize.Cell{Value: "(6а)"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "(6б)"},
		excelize.Cell{},
		excelize.Cell{Value: "Табличная часть УПД"},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"В пути\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Доставлен\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Принят\""},
		excelize.Cell{},
	})

	streamWriter.MergeCell("A1", "B1")
	streamWriter.MergeCell("G1", "H1")
	streamWriter.MergeCell("I1", "J1")
	streamWriter.MergeCell("O1", "P1")
	streamWriter.MergeCell("Q1", "Y1")
	streamWriter.MergeCell("AA1", "AB1")
	streamWriter.MergeCell("AC1", "AD1")
	streamWriter.MergeCell("AE1", "AF1")

	streamWriter.SetRow("A2", []interface{}{
		excelize.Cell{Value: "Номер заказа"},
		excelize.Cell{Value: "Дата заказа"},
		excelize.Cell{Value: "Дата согласования"},
		excelize.Cell{Value: "Продавец"},
		excelize.Cell{Value: "Адрес"},
		excelize.Cell{Value: "ИНН/КПП продавца"},
		excelize.Cell{Value: "Грузоотправитель"},
		excelize.Cell{Value: "Адрес грузоотправителя"},
		excelize.Cell{Value: "Грузополучатель"},
		excelize.Cell{Value: "Адрес грузополучателя"},
		excelize.Cell{Value: "Покупатель"},
		excelize.Cell{Value: "Адрес покупателя"},
		excelize.Cell{Value: "ИНН поставщика"},
		excelize.Cell{Value: "КПП поставщика"},
		excelize.Cell{Value: "ИНН покупателя"},
		excelize.Cell{Value: "КПП покупателя"},
		excelize.Cell{Value: "Код товара/работ, услуг"},
		excelize.Cell{Value: "Наименование товара"},
		excelize.Cell{Value: "Единица обозначения - условное обозначение (национальное)"},
		excelize.Cell{Value: "Количество (объём)"},
		excelize.Cell{Value: "Цена (тариф) за единицу измерения"},
		excelize.Cell{Value: "Стоимость товаров (работ, услуг), имущественных прав без налога - всего"},
		excelize.Cell{Value: "Налоговая ставка"},
		excelize.Cell{Value: "Сумма налога, предъявляемая покупателю"},
		excelize.Cell{Value: "Стоимость товаров (работ, услуг), имущественных прав с налогом - всего"},
		excelize.Cell{Value: "Дата поставки (предполагаемая)"},
		excelize.Cell{Value: "Дата отгрузки"},
		excelize.Cell{Value: "Время отгрузки"},
		excelize.Cell{Value: "Дата передачи"},
		excelize.Cell{Value: "Время передачи"},
		excelize.Cell{Value: "Дата приёмки"},
		excelize.Cell{Value: "Время приёмки"},
		excelize.Cell{Value: "Статус"},
	})

	userInfo := mh.auth.getUserInfo(r)
	orders, err := mh.db.getOrders(r.Context(), userInfo, ordersFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	row := 3
	for _, order := range orders {
		orderStartRow := row

		orderDetails, err := mh.db.getOrder(r.Context(), userInfo, order["id"].(int))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if len(orderDetails) == 0 {
			continue
		}

		orderDetail := orderDetails[0]
		sum := orderDetail["sum"].(float64)
		nds := orderDetail["nds"].(float64)
		tax := math.Floor(sum*nds) / 100

		streamWriter.SetRow(fmt.Sprintf("A%v", row), []interface{}{
			excelize.Cell{Value: order["contractor_number"]},
			excelize.Cell{Value: toDate(order["ordered_date"])},
			excelize.Cell{Value: toDate(order["closed_date"])},
			excelize.Cell{Value: "Общество с ограниченной ответственностью \"Центр Промышленных Закупок\""},
			excelize.Cell{Value: "127299, г. Москва, ул. Клары Цеткин, д. 2, помещ. 138"},
			excelize.Cell{Value: "3528136252/771301001"},
			excelize.Cell{Value: order["seller_name"]},
			excelize.Cell{Value: order["seller_address"]},
			excelize.Cell{Value: order["customer_name"]},
			excelize.Cell{Value: order["customer_address"]},
			excelize.Cell{Value: order["customer_name"]},
			excelize.Cell{Value: order["customer_address"]},
			excelize.Cell{Value: order["seller_inn"]},
			excelize.Cell{Value: order["seller_kpp"]},
			excelize.Cell{Value: order["customer_inn"]},
			excelize.Cell{Value: order["customer_kpp"]},

			excelize.Cell{Value: orderDetail["product_id"]},
			excelize.Cell{Value: orderDetail["name"]},
			excelize.Cell{Value: "Шт"},
			excelize.Cell{Value: orderDetail["count"]},
			excelize.Cell{Value: orderDetail["price"]},
			excelize.Cell{Value: sum},
			excelize.Cell{Value: nds},
			excelize.Cell{Value: tax},
			excelize.Cell{Value: tax + sum},

			excelize.Cell{Value: toDate(order["shipping_date_est"])},
			excelize.Cell{Value: toDate(order["shipped_date"])},
			excelize.Cell{Value: toTime(order["shipped_date"])},
			excelize.Cell{Value: toDate(order["delivered_date"])},
			excelize.Cell{Value: toTime(order["delivered_date"])},
			excelize.Cell{Value: toDate(order["accepted_date"])},
			excelize.Cell{Value: toTime(order["accepted_date"])},
			excelize.Cell{Value: order["status"]},
		})
		row++

		for i := 1; i < len(orderDetails); i++ {
			orderDetail = orderDetails[i]

			sum := orderDetail["sum"].(float64)
			nds := orderDetail["nds"].(float64)
			tax := math.Floor(sum*nds) / 100

			streamWriter.SetRow(fmt.Sprintf("Q%v", row), []interface{}{
				excelize.Cell{Value: orderDetail["product_id"]},
				excelize.Cell{Value: orderDetail["name"]},
				excelize.Cell{Value: "Шт"},
				excelize.Cell{Value: orderDetail["count"]},
				excelize.Cell{Value: orderDetail["price"]},
				excelize.Cell{Value: sum},
				excelize.Cell{Value: nds},
				excelize.Cell{Value: tax},
				excelize.Cell{Value: tax + sum},
			})

			row++
		}

		if len(orderDetails) > 1 {

			for col := 'A'; col <= 'P'; col++ {
				err = streamWriter.MergeCell(fmt.Sprintf("%c%v", col, orderStartRow), fmt.Sprintf("%c%v", col, row-1))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
			}

			for _, col := range []string{"Z", "AA", "AB", "AC", "AD", "AE", "AF", "AG"} {
				err = streamWriter.MergeCell(fmt.Sprintf("%s%v", col, orderStartRow), fmt.Sprintf("%s%v", col, row-1))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
			}
		}

	}

	err = streamWriter.Flush()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	xls.SaveAs(file.Name())

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("Заказы.xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file.Name())

	return nil
}
