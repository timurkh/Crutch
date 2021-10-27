package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	methods *MethodHandlers
	err     error
)

func TestInit(t *testing.T) {
	log.SetLevel(logrus.ErrorLevel)
	methods, _, err = initAuthMethodHandlers()

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestSearchProducts(t *testing.T) {
	t.Run("Ищем Денисом (Северсталь) \"Ключ гаечный рожковый односторонний VDE 1000V 10 мм\" в Оленегорске", func(t *testing.T) {

		userInfo := UserInfo{Id: 7}

		searchQuery := SearchQuery{
			Text:   "Ключ гаечный рожковый односторонний VDE 1000V 10 мм",
			CityID: 703,
		}
		sr, err, _ := methods.searchProducts(context.Background(), userInfo, searchQuery)
		if err != nil {
			t.Errorf("Search failed - %v", err)
		}

		if sr.Results[0].Code != "VDA-PE010" {
			t.Errorf("Found wrong product - %v (id %v) instead of VDA-PE010", sr.Results[0].Code, sr.Results[0].Id)
		}
	})

	t.Run("Ищем Виталием (Гарвин) \"УОНИ-13/55 4,0 мм 5кг\" в Череповце", func(t *testing.T) {

		userInfo := UserInfo{Id: 14, SupplierId: 5}

		searchQuery := SearchQuery{
			Text:   "УОНИ-13/55 4,0 мм 5кг",
			CityID: 1042,
		}
		sr, err, _ := methods.searchProducts(context.Background(), userInfo, searchQuery)
		if err != nil {
			t.Errorf("Search failed - %v", err)
		}

		if sr.Results[0].Code != "ЯрЭМП-УОНИ-13/55Ф4" || sr.Results[0].Supplier != "ООО \"Гарвин Индастриал\"" {
			t.Errorf("Found wrong product - %v (id %v, supplier %v) instead of ЯрЭМП-УОНИ-13/55Ф4", sr.Results[0].Code, sr.Results[0].Id, sr.Results[0].Supplier)
		}
	})
}

func TestAPI(t *testing.T) {
	userInfo := UserInfo{
		Id:           14,
		SupplierId:   5,
		Email:        "test@supplier.ru",
		CompanyAdmin: true,
	}

	t.Run("Включаем API для тестового пользователя поставщика", func(t *testing.T) {

		apiCreds, err, _ := methods.getApiCredentials(context.Background(), userInfo)
		if err != nil {
			t.Errorf("Failed to get API credentials - %v", err)
		}

		b := false
		params := apiCredParams{}
		params.Enabled = &b
		newCreds, err, _ := methods.putApiCredentials(context.Background(), userInfo, params)

		if err != nil {
			t.Errorf("Failed to update API credentials - %v", err)
		}

		if newCreds.Enabled != false {
			t.Errorf("API is not disabled")
		}

		b = true
		params.Enabled = &b
		params.Password = &b
		newCreds, err, _ = methods.putApiCredentials(context.Background(), userInfo, params)

		if err != nil {
			t.Errorf("Failed to update API credentials - %v", err)
		}

		if newCreds.Password != apiCreds.Password {
			t.Errorf("Password was not updated")
		}

		if newCreds.Enabled != true {
			t.Errorf("API is not enabled")
		}
	})

	t.Run("Проверяем /orders/", func(t *testing.T) {

		layout := "2006-01-02T15:04:05.000Z"
		ts, _ := time.Parse(layout, "2021-09-01T07:00:00.000Z")
		te, err := time.Parse(layout, "2021-09-30T21:00:00.000Z")
		ordersFilter := OrdersFilter{
			Start:        ts,
			End:          te,
			DateColumn:   "date_closed",
			ItemsPerPage: 10,
			Page:         0,
		}

		orders, err, _ := methods.getOrders(context.Background(), userInfo, ordersFilter)
		if err != nil {
			t.Errorf("Failed to get list of orders - %v", orders)
		}

		if orders.Sum != 654200.95 || orders.Count != 49 || orders.SumWithTax != 785041.15 {
			t.Errorf("Got wrong orders sum %v count %v sum_with_tax %v", orders.Sum, orders.Count, orders.SumWithTax)
		}

		dt, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", "2021-10-26T10:03:26.738252+03:00")
		if len(orders.Orders) != 10 || orders.Orders[0].Id != 969 || *orders.Orders[0].DeliveredDate != dt {
			t.Errorf("Got wrong orders data - %v orders, %v orders[0].id, %v orders[0].delivered_date != %v", len(orders.Orders), orders.Orders[0].Id, *orders.Orders[0].DeliveredDate, dt)
		}
	})

}

func TestExcel(t *testing.T) {

	userInfo := UserInfo{
		Id:            464,
		Staff:         true,
		CanReadOrders: true,
	}

	layout := "2006-01-02T15:04:05.000Z"
	ts, _ := time.Parse(layout, "2021-07-02T07:00:00.000Z")
	te, _ := time.Parse(layout, "2021-07-02T21:00:00.000Z")
	ordersFilter := OrdersFilter{
		Start:            ts,
		End:              te,
		DateColumn:       "date_closed",
		SelectedStatuses: []int{},
	}

	t.Run("Проверяем выгрузку заказов", func(t *testing.T) {

		fileName := "./test_files/orders.xls"
		defer os.Remove(fileName)
		err, _ := methods.getOrdersExcel(context.Background(), userInfo, ordersFilter, fileName)
		if err != nil {
			t.Errorf("Failed to export orders to excel - %v", err)
		}

		file1, err := os.Stat(fileName)
		file2, err := os.Stat("./test_files/orders.xls.golden")

		if file1.Size() != file2.Size() {
			t.Errorf("Generated xls file is different from golden image")
		}
	})
}
