package main

import (
	"context"
	"fmt"

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
	first_name   string
	last_name    string
	email        string
	is_superuser bool
	verified     bool
	blocked      bool
}

func (db *DBHelper) getUserInfo(userId int) (*UserDBInfo, error) {
	ui := UserDBInfo{}
	err := db.pool.QueryRow(context.Background(), "select first_name, last_name, email, is_superuser, verified, blocked from core_user where id=$1", userId).Scan(&ui.first_name, &ui.last_name, &ui.email, &ui.is_superuser, &ui.verified, &ui.blocked)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user info: %v", err)
	}

	return &ui, nil

}

func (db *DBHelper) getUserConsigneeCities(ctx context.Context, userId int) (cities []City, err error) {
	rows, _ := db.pool.Query(ctx, "select distinct city_id, city from core_user cu join core_user_contractors cuc on cu.id = cuc.user_id join consignee_consignee con using(contractor_id) join company_city com on com.id = con.city_id where cuc.user_id=$1", userId)

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

func (db *DBHelper) getProductEntries(ctx context.Context, product_ids []int, user_id int, city_id int) (products []map[string]string, err error) {

	client_cities := `
	SELECT DISTINCT city_id 
	FROM core_user_contractors cuc 
		JOIN consignee_consignee con USING(contractor_id) 
		JOIN company_city com on com.id = con.city_id 
	WHERE cuc.user_id=$2
		AND ($3 = 0 OR city_id = $3)`

	supplier_warehouses := `
	SELECT sw.id 
	FROM supplier_warehouse sw 
		INNER JOIN supplier_warehouse_delivery_cities swc ON (sw.id = swc.warehouse_id) 
	WHERE sw.is_visible = true  
		AND swc.city_id IN (` + client_cities + `)`

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
		AND pr.warehouse_id IN (` + supplier_warehouses + `)
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
	ORDER BY ordering`, product_ids, user_id, city_id)

	products = make([]map[string]string, 0)
	for rows.Next() {
		var category, name, code, description, supplier string
		var price float64
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
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

		entry := map[string]string{
			"id":          fmt.Sprintf("%v", values[0]),
			"category":    category,
			"code":        code,
			"name":        name,
			"description": description,
			"rest":        fmt.Sprintf("%v", values[5]),
			"price":       fmt.Sprintf("%v", price),
			"supplier":    supplier,
		}
		products = append(products, entry)
	}

	return products, rows.Err()
}
