package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"unicode/utf8"

	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/xuri/excelize/v2"
)

const itemsPerPage = 200

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

		if len(cities) == 0 && userInfo.SupplierId != 0 {
			cities, err = mh.db.getSupplierCities(r.Context(), userInfo.SupplierId)
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

	log.Info(fmt.Printf("Handling search request text=%s, category=%s, code=%s, name=%s, property=%s, page=%v\n", searchQuery.Text, searchQuery.Category, searchQuery.Code, searchQuery.Name, searchQuery.Property, searchQuery.Page))
	totalPages := 0
	entries := make([]map[string]interface{}, 0)
	tries := 0
	for {
		tries++

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

		if len(entries) > itemsPerPage/10 || searchQuery.Page >= totalPages-1 {
			break
		}
		searchQuery.Page++
	}

	log.Info("Have done ", tries, " queries to get ", len(entries), " product entries")

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

	log.Debug("Got info for ", len(products), " entries")

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

	if !userInfo.Admin && !userInfo.Staff {
		log.Error("Insufficient privileges to get list of counterparts")
		http.Error(w, "Inssuficient privileges", http.StatusUnauthorized)
		return err
	}

	log.Info("Getting list of counterparts, filter ", filter)

	counterparts, err := mh.db.getCounterparts(r.Context(), userInfo, filter)
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

func (mh *MethodHandlers) getCounterpartsExcelHandler(w http.ResponseWriter, r *http.Request) error {

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

	if !userInfo.Admin && !userInfo.Staff {
		log.Error("Insufficient privileges to get excel with list of counterparts")
		http.Error(w, "This resource requires admin privileges", http.StatusUnauthorized)
		return err
	}

	log.Info("Getting list of counterparts, filter ", filter)

	counterparts, err := mh.db.getCounterparts(r.Context(), userInfo, filter)
	if err != nil {
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

	columnNames := []interface{}{
		excelize.Cell{Value: "#"},
		excelize.Cell{Value: "Название компании"},
		excelize.Cell{Value: "Роль"},
		excelize.Cell{Value: "ИНН"},
		excelize.Cell{Value: "КПП"},
		excelize.Cell{Value: "ОГРН"},
		excelize.Cell{Value: "Юридический адрес"},
		excelize.Cell{Value: "Фактический адрес"},
		excelize.Cell{Value: "Директор"},
		excelize.Cell{Value: "Контактное лицо"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "Банк"},
		excelize.Cell{Value: "БИК"},
		excelize.Cell{Value: "Корр. счёт"},
		excelize.Cell{Value: "Рассчётный счёт"},
		excelize.Cell{Value: "Дополнительные поля"},
		excelize.Cell{Value: "Телефон банка"},
		excelize.Cell{Value: "Счёт"},
		excelize.Cell{Value: "IBAN"},
		excelize.Cell{Value: "SWIFT"},
		excelize.Cell{Value: "Страна"},
		excelize.Cell{Value: "Город"},
		excelize.Cell{Value: "E-mail"},
		excelize.Cell{Value: "Сайт"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "Имя"},
		excelize.Cell{Value: "Телефон"},
		excelize.Cell{Value: "E-Mail"},
	}
	for i, columnName := range columnNames {
		cellWidth := utf8.RuneCountInString(columnName.(excelize.Cell).Value.(string)) + 2 // + 2 for margin
		if cellWidth < 6 {
			cellWidth = 6
		}
		streamWriter.SetColWidth(i+1, i+1, float64(cellWidth))
	}

	streamWriter.MergeCell("A1", "T1")
	streamWriter.SetRow("U1", []interface{}{
		excelize.Cell{Value: "Поставщик"},
	})

	streamWriter.MergeCell("U1", "Y1")
	streamWriter.SetRow("Z1", []interface{}{
		excelize.Cell{Value: "Администратор"},
	})

	streamWriter.MergeCell("Z1", "AB1")

	streamWriter.SetRow("A2", columnNames)

	row := 3
	for _, cp := range counterparts {

		streamWriter.SetRow(fmt.Sprintf("A%v", row), []interface{}{
			excelize.Cell{Value: cp["id"]},
			excelize.Cell{Value: cp["name"]},
			excelize.Cell{Value: cp["role"]},
			excelize.Cell{Value: cp["inn"]},
			excelize.Cell{Value: cp["kpp"]},
			excelize.Cell{Value: cp["ogrn"]},
			excelize.Cell{Value: cp["address"]},
			excelize.Cell{Value: cp["actual_address"]},
			excelize.Cell{Value: cp["director_name"]},
			excelize.Cell{Value: cp["contact_name"]},
			excelize.Cell{Value: cp["phone"]},
			excelize.Cell{Value: cp["bank"]},
			excelize.Cell{Value: cp["bik"]},
			excelize.Cell{Value: cp["corr_account"]},
			excelize.Cell{Value: cp["pay_account"]},
			excelize.Cell{Value: cp["extra_data"]},
			excelize.Cell{Value: cp["bank_phone"]},
			excelize.Cell{Value: cp["account"]},
			excelize.Cell{Value: cp["IBAN"]},
			excelize.Cell{Value: cp["SWIFT"]},
			excelize.Cell{Value: cp["country"]},
			excelize.Cell{Value: cp["city"]},
			excelize.Cell{Value: cp["seller_email"]},
			excelize.Cell{Value: cp["seller_site"]},
			excelize.Cell{Value: cp["seller_phone"]},
		})

		if cp["admins"] != nil {
			admins := cp["admins"].(map[string]interface{})
			rowStart := row
			for _, a := range admins {
				admin := a.(map[string]interface{})

				streamWriter.SetRow(fmt.Sprintf("Z%v", row), []interface{}{
					excelize.Cell{Value: toString(admin["name"])},
					excelize.Cell{Value: toString(admin["phone"])},
					excelize.Cell{Value: toString(admin["email"])},
				})
				row++
			}

			adminsLen := len(admins)

			if adminsLen > 1 {

				for col := 'A'; col <= 'Y'; col++ {
					streamWriter.MergeCell(fmt.Sprintf("%c%v", col, rowStart), fmt.Sprintf("%c%v", col, row-1))
				}
			}
		} else {

			row++
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

	if ordersFilter.Page == 0 {
		count, sum, e := mh.db.getOrdersSum(r.Context(), userInfo, ordersFilter)
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return e
		}
		err = json.NewEncoder(w).Encode(struct {
			Orders interface{} `json:"orders"`
			Count  interface{} `json:"count"`
			Sum    interface{} `json:"sum"`
		}{orders, count, sum})

	} else {
		err = json.NewEncoder(w).Encode(struct {
			Orders interface{} `json:"orders"`
		}{orders})
	}

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

func getNums(orderDetail map[string]interface{}) (float64, float64, float64, float64, float64) {
	count := orderDetail["count"].(float64)
	price := orderDetail["price"].(float64)
	sum := orderDetail["sum"].(float64)
	nds := orderDetail["nds"].(float64)
	tax := math.Round(sum*nds) / 100

	return count, price, sum, nds, tax
}

func (mh *MethodHandlers) getOrdersExcelHandler(w http.ResponseWriter, r *http.Request) error {

	var ordersFilter OrdersFilter
	err := schema.NewDecoder().Decode(&ordersFilter, r.URL.Query())
	if err != nil {
		err = fmt.Errorf("Failed to decode filter: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	ordersFilter.Page = 0
	ordersFilter.ItemsPerPage = 0

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

	//set column width
	columnNames := []interface{}{
		excelize.Cell{Value: "ID"},
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
		excelize.Cell{Value: "Артикул"},
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
	}
	for i, columnName := range columnNames {
		cellWidth := utf8.RuneCountInString(columnName.(excelize.Cell).Value.(string)) + 2 // + 2 for margin
		streamWriter.SetColWidth(i+1, i+1, float64(cellWidth))
	}

	streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{},
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
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"В пути\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Доставлен\""},
		excelize.Cell{},
		excelize.Cell{Value: "Статус заказа \"Принят\""},
		excelize.Cell{},
	})

	streamWriter.MergeCell("B1", "C1")
	streamWriter.MergeCell("H1", "I1")
	streamWriter.MergeCell("J1", "K1")
	streamWriter.MergeCell("P1", "Q1")
	streamWriter.MergeCell("R1", "AA1")
	streamWriter.MergeCell("AC1", "AD1")
	streamWriter.MergeCell("AE1", "AF1")
	streamWriter.MergeCell("AG1", "AH1")

	streamWriter.SetRow("A2", columnNames)

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

		for i := 0; i < len(orderDetails); i++ {
			orderDetail := orderDetails[i]

			count, price, sum, nds, tax := getNums(orderDetail)

			streamWriter.SetRow(fmt.Sprintf("A%v", row), []interface{}{
				excelize.Cell{Value: order["id"]},
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
				excelize.Cell{Value: orderDetail["code"]},
				excelize.Cell{Value: orderDetail["name"]},
				excelize.Cell{Value: "Шт"},
				excelize.Cell{Value: count},
				excelize.Cell{Value: price},
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
		}

		if len(orderDetails) > 1 {

			for col := 'A'; col <= 'Q'; col++ {
				err = streamWriter.MergeCell(fmt.Sprintf("%c%v", col, orderStartRow), fmt.Sprintf("%c%v", col, row-1))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
			}

			for _, col := range []string{"AB", "AC", "AD", "AE", "AF", "AG", "AH", "AI"} {
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

func (mh *MethodHandlers) getOrdersCSVHandler(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)

	file, err := os.CreateTemp("/tmp", "*.csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	defer os.Remove(file.Name())

	mh.db.getOrdersCSV(r.Context(), userInfo, file)

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("Заказы.csv"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, file.Name())

	return nil
}

func (mh *MethodHandlers) getCurrentUserSI(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)
	cities, err := mh.db.getUserConsigneeCities(r.Context(), userInfo)

	if len(cities) == 0 && userInfo.SupplierId != 0 {
		cities, err = mh.db.getSupplierCities(r.Context(), userInfo.SupplierId)
	}

	compareCount, err := mh.db.getCompareItemsCount(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		UserInfo
		Cities            []City `json:"cities"`
		CompareItemsCount int    `json:"compareItemsCount"`
	}{userInfo, cities, compareCount})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}

func (mh *MethodHandlers) getCartContent(w http.ResponseWriter, r *http.Request) error {

	userInfo := mh.auth.getUserInfo(r)
	cartNumbers, err := mh.db.getCartNumbers(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	cartItems, err := mh.db.getCartItems(r.Context(), userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		CartNumbers
		CartItems map[int]CartItem `json:"cartItems"`
	}{*cartNumbers, cartItems})

	if err != nil {
		err = fmt.Errorf("Error while preparing json reponse: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil

}
