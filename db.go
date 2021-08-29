package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DBHelper struct {
	pool *pgxpool.Pool
}

func initDBHelper(host string, user string, password string, database string) (*DBHelper, error) {
	url := "postgres://" + user + ":" + password + "@" + host + "/" + database
	conf, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	conf.ConnConfig.Logger = logrusadapter.NewLogger(log)
	conf.ConnConfig.LogLevel = pgx.LogLevelTrace

	pool, err := pgxpool.ConnectConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	db := DBHelper{pool}
	return &db, nil
}

type UserDBInfo struct {
	first_name      string
	last_name       string
	email           string
	is_superuser    bool
	can_read_orders bool
	verified        bool
	blocked         bool
}

func (db *DBHelper) getUserInfo(userId int) (*UserDBInfo, error) {
	ui := UserDBInfo{}
	err := db.pool.QueryRow(context.Background(), `
		SELECT first_name, 
			last_name, 
			email, 
			is_superuser, 
			COALESCE( staff_can_read_orders, FALSE) staff_can_read_orders,
			verified, 
			blocked 
		FROM 
			core_user cu 
			LEFT JOIN (
				SELECT 
					user_id, 
					TRUE as staff_can_read_orders 
				FROM core_user_user_permissions up 
				WHERE permission_id=1067) ro 
			ON (ro.user_id=cu.id AND cu.is_staff) 
		WHERE cu.id =$1`, userId).Scan(&ui.first_name, &ui.last_name, &ui.email, &ui.is_superuser, &ui.can_read_orders, &ui.verified, &ui.blocked)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user info: %v", err)
	}

	return &ui, nil

}

func (db *DBHelper) getUserConsigneeCities(ctx context.Context, userInfo UserInfo) (cities []City, err error) {

	var rows pgx.Rows

	if userInfo.Admin {
		rows, _ = db.pool.Query(ctx, "SELECT distinct id, city FROM company_city")
	} else {
		rows, _ = db.pool.Query(ctx, "SELECT distinct city_id, city FROM core_user cu JOIN core_user_contractors cuc on cu.id = cuc.user_id join consignee_consignee con using(contractor_id) join company_city com on com.id = con.city_id where cuc.user_id=$1", userInfo.Id)
	}

	cities = make([]City, 0)
	for rows.Next() {
		var city City
		err := rows.Scan(&city.Id, &city.Name)
		if err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}

	return cities, rows.Err()
}

func (db *DBHelper) getProductEntries(ctx context.Context, product_ids []int, products_score map[int]float64, userInfo UserInfo, city_id int) (products []map[string]interface{}, err error) {

	args := []interface{}{product_ids}

	supplier_warehouses := ""
	if city_id > 0 || !userInfo.Admin {
		client_cities := `
		SELECT DISTINCT city_id 
		FROM core_user_contractors cuc 
			JOIN consignee_consignee con USING(contractor_id) 
			JOIN company_city com on com.id = con.city_id 
		WHERE cuc.user_id=$2`

		args = append(args, userInfo.Id)

		if city_id > 0 {
			client_cities = client_cities + " AND city_id = $3"
			args = append(args, city_id)
		}

		supplier_warehouses = `
		AND pr.warehouse_id IN (
		SELECT sw.id 
		FROM supplier_warehouse sw 
			INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
		WHERE sw.is_visible = true  
			AND swc.city_id IN (` + client_cities + `))`
	}

	product_quantity := `
	SELECT
		pp.id,
		SUM(pr.rest) AS rest,
		ordering
	FROM
		product_product pp
		JOIN product_modification pm ON ( pp.id = pm.product_id )
		JOIN product_rest  pr ON ( pm.id = pr.modification_id )
		JOIN (SELECT * FROM unnest($1::int[]) WITH ORDINALITY) x (id, ordering) ON (pp.id = x.id)
	WHERE
		pp.deleted = false
		AND pp.is_reference = false
		AND pp.b_placement_state = 'placed'
		AND pp.category_id IS NOT NULL
		AND pp.hidden = false  
	` + supplier_warehouses + `
	GROUP BY pp.id, ordering`

	rows, _ := db.pool.Query(ctx, `
	SELECT
		pp.id,
		pc.name as category_name,
		pp.name,
		pp.code,
		pp.description,
		pr.rest,
		pp.product_price,
		cc.name as supplier
	FROM 
		( `+product_quantity+` 
			) pr 
		JOIN product_product pp USING (id) 
		JOIN product_category pc ON ( pp.category_id = pc.id )
		JOIN product_suppliercategory psc ON (pp.supplier_category_id = psc.id)
		JOIN company_company cc ON (cc.object_id=supplier_id AND content_type_id=186)
	WHERE 
		pc.hidden = false
		AND (pp.enable_preorder = true OR NOT pr.rest = 0.0)
	ORDER BY ordering`, args...)

	products = make([]map[string]interface{}, 0)
	for rows.Next() {
		var category, name, code, description, supplier string
		var price float64
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		id := int(values[0].(int32))

		if values[1] != nil {
			category = values[1].(string)
		}
		name = values[2].(string)
		if values[3] != nil {
			code = values[3].(string)
		}
		if values[4] != nil {
			description = values[4].(string)
		}

		p := values[6].(pgtype.Numeric)
		p.AssignTo(&price)

		if values[7] != nil {
			supplier = values[7].(string)
		}

		entry := map[string]interface{}{
			"id":          id,
			"category":    category,
			"code":        code,
			"name":        name,
			"description": description,
			"rest":        fmt.Sprintf("%v", values[5]),
			"price":       fmt.Sprintf("%v", price),
			"supplier":    supplier,
		}
		entry["score"] = products_score[id]
		products = append(products, entry)
	}

	return products, rows.Err()
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	return v.(float64)
}

func (db *DBHelper) ordersAccessRightsFilter(userInfo UserInfo, args []interface{}) (string, []interface{}) {
	filterUsers := ""
	if !userInfo.Admin && !userInfo.CanReadOrders {
		filterUsers = ` AND oo.user_id in (
		SELECT ou.id 
		FROM core_user ou 
			JOIN core_user cu ON (ou.lft < cu.rght  AND ou.lft > cu.lft  AND ou.tree_id = cu.tree_id) 
		WHERE cu.id=$1)
		AND FALSE` //temporary disable for non staff
		args = append(args, userInfo.Id)
	}
	return filterUsers, args
}

func (db *DBHelper) getOrders(ctx context.Context, userInfo UserInfo) (orders []map[string]interface{}, err error) {

	args := make([]interface{}, 0)
	filterUsers, args := db.ordersAccessRightsFilter(userInfo, args)

	queryOrders := `
		SELECT oo.id, 
			ov.order_sum,
			os.status,
			oc.date_ordered,
			oo.date_closed,
			oo.shipping_date,
			seller.object_id as seller_id,
			seller.name AS seller_name,
			seller.inn AS seller_inn,
			seller.kpp AS seller_kpp,
			seller.jur_address AS seller_address,
			cu.id AS buyer_id,
			cu.last_name || ' ' || cu.first_name || ' ' || cu.middle_name AS buyer,
			customer.object_id AS customer_id,
			customer.name AS customer_name,
			customer.inn AS customer_inn,
			customer.kpp AS customer_kpp,
			customer.jur_address AS customer_address,
			cc.name as consignee_name,
			oo.on_order_coupon,
			oo.on_order_coupon_fixed,
			oo.contractor_number
		FROM order_order oo 
			JOIN (
				SELECT oo.id, 
					round(((((sum((oi.count * ((((oi.item_price - oi.coupon_fixed) * ((100)::numeric - oi.coupon_percent)) / (100)::numeric))::double precision)) * (((100)::numeric - oo.on_order_coupon))::double precision) / (100)::double precision) - (oo.on_order_coupon_fixed)::double precision))::numeric, 2) AS order_sum
				FROM order_order oo 
					LEFT JOIN order_orderitem oi ON (oo.id = oi.order_id)
				GROUP BY oo.id
			) ov USING (id)
			JOIN (
				SELECT object_id_int AS order_id, MIN(rr.date_created) AS date_ordered
				FROM reversion_version rv JOIN reversion_revision rr ON rv.revision_id = rr.id 
				WHERE content_type_id=115 and serialized_data::json->0->'fields'->>'status' = '1'
				GROUP BY object_id_int) oc ON oc.order_id = oo.id 
			JOIN order_orderstatus os ON (oo.status_id = os.id)
			JOIN company_company seller ON (seller.object_id=oo.supplier_id AND seller.content_type_id=186)
			JOIN core_user cu ON (cu.id = oo.user_id)
			LEFT JOIN consignee_consignee cc ON (cc.id = oo.consignee_id)
			JOIN company_company customer ON (customer.object_id=oo.contractor_id AND customer.content_type_id=79)
		WHERE oo.status_id NOT IN (17, 18)`
	queryOrders = queryOrders + filterUsers + `	ORDER BY oc.date_ordered DESC`
	rows, _ := db.pool.Query(ctx, queryOrders, args...)

	orders = make([]map[string]interface{}, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		id := int(values[0].(int32))
		var sum float64
		if values[1] != nil {
			s := values[1].(pgtype.Numeric)
			s.AssignTo(&sum)
		}

		status := values[2].(string)
		var date_ordered, date_closed, shipping_date string
		if values[3] != nil {
			date_ordered = values[3].(time.Time).Format("2006-01-02")
		}
		if values[4] != nil {
			date_closed = values[4].(time.Time).Format("2006-01-02")
		}
		if values[5] != nil {
			shipping_date = values[5].(time.Time).Format("2006-01-02")
		}
		seller_id := int(values[6].(int32))
		seller_name := toString(values[7])
		seller_inn := toString(values[8])
		seller_kpp := toString(values[9])
		seller_address := toString(values[10])
		buyer_id := int(values[11].(int32))
		buyer := toString(values[12])
		customer_id := int(values[13].(int32))
		customer_name := toString(values[14])
		customer_inn := toString(values[15])
		customer_kpp := toString(values[16])
		customer_address := toString(values[17])
		consignee_name := toString(values[18])

		var on_order_coupon, on_order_coupon_fixed float64
		if values[19] != nil {
			s := values[19].(pgtype.Numeric)
			s.AssignTo(&on_order_coupon)
		}

		if values[20] != nil {
			s := values[20].(pgtype.Numeric)
			s.AssignTo(&on_order_coupon_fixed)
		}

		s := "000000000"
		s = s + toString(values[21])
		contractor_number := s[len(s)-10:]

		entry := map[string]interface{}{
			"id":                    id,
			"contractor_number":     contractor_number,
			"sum":                   sum,
			"status":                status,
			"ordered_date":          date_ordered,
			"closed_date":           date_closed,
			"shipping_date":         shipping_date,
			"seller_id":             seller_id,
			"seller_name":           seller_name,
			"seller_inn":            seller_inn,
			"seller_kpp":            seller_kpp,
			"seller_address":        seller_address,
			"buyer_id":              buyer_id,
			"buyer":                 buyer,
			"customer_id":           customer_id,
			"customer_name":         customer_name,
			"customer_inn":          customer_inn,
			"customer_kpp":          customer_kpp,
			"customer_address":      customer_address,
			"consignee_name":        consignee_name,
			"on_order_coupon":       on_order_coupon,
			"on_order_coupon_fixed": on_order_coupon_fixed,
		}
		orders = append(orders, entry)
	}

	return orders, rows.Err()
}

func (db *DBHelper) getOrder(ctx context.Context, userInfo UserInfo, orderId int) (orders []map[string]interface{}, err error) {

	args := []interface{}{orderId}
	filterUsers, args := db.ordersAccessRightsFilter(userInfo, args)

	queryOrderDetails := `
		SELECT 
			pp.id,
			pp.name,
			pp.code,
			sw.name AS warehouse,
			oi.count, 
			oi.item_price, 
			oi.rate_nds, 
			oi.coupon_percent, 
			oi.coupon_fixed, 
			oi.coupon_value, 
			oi.comment 
		FROM order_orderitem oi
			JOIN order_order oo ON (oi.order_id = oo.id)
			JOIN product_modification pm ON (oi.modification_id = pm.id)
			JOIN product_product pp ON (pm.product_id = pp.id)
			LEFT JOIN supplier_warehouse sw ON (sw.id = oi.warehouse_id)
		WHERE oo.id=$1`
	queryOrderDetails = queryOrderDetails + filterUsers + `	ORDER BY oi.id DESC`
	rows, _ := db.pool.Query(ctx, queryOrderDetails, args...)

	orderDetails := make([]map[string]interface{}, 0)
	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		id := int(values[0].(int32))
		name := toString(values[1])
		code := toString(values[2])
		warehouse := toString(values[3])
		count := values[4].(float64)
		var price float64
		if values[5] != nil {
			s := values[5].(pgtype.Numeric)
			s.AssignTo(&price)
		}
		nds := toFloat(values[6])
		var coupon_percent float64
		if values[7] != nil {
			s := values[7].(pgtype.Numeric)
			s.AssignTo(&coupon_percent)
		}
		var coupon_fixed float64
		if values[8] != nil {
			s := values[8].(pgtype.Numeric)
			s.AssignTo(&coupon_fixed)
		}
		var coupon_value float64
		if values[9] != nil {
			s := values[9].(pgtype.Numeric)
			s.AssignTo(&coupon_value)
		}
		comment := toString(values[10])

		entry := map[string]interface{}{
			"id":             id,
			"name":           name,
			"code":           code,
			"warehouse":      warehouse,
			"count":          count,
			"price":          price,
			"nds":            nds,
			"coupon_percent": coupon_percent,
			"coupon_fixed":   coupon_fixed,
			"coupon_value":   coupon_value,
			"comment":        comment,
		}
		orderDetails = append(orderDetails, entry)
	}

	return orderDetails, rows.Err()
}
