package jobs

import (
	"database/sql"
	"fmt"
	"log"
	"neptune/check_jobs/slack_utils"
	"os"
	"time"
	_ "github.com/lib/pq"
)

type DbConfig struct {
	User     string
	Database string
	Password string
	Host     string
	Port     string
}

func CheckAvgDbCounts(channelID string) {
	hireableConfig := DbConfig{
		User:     os.Getenv("POSTGRES_USER_HIR"),
		Database: os.Getenv("POSTGRES_DATABASE_HIR"),
		Password: os.Getenv("POSTGRES_PASSWORD_HIR"),
		Host:     os.Getenv("POSTGRES_HOST_HIR"),
		Port:     os.Getenv("POSTGRES_PORT_HIR"),
	}
	wallaConfig := DbConfig{
		User:     os.Getenv("POSTGRES_USER_WALLA"),
		Database: os.Getenv("POSTGRES_DATABASE_WALLA"),
		Password: os.Getenv("POSTGRES_PASSWORD_WALLA"),
		Host:     os.Getenv("POSTGRES_HOST_WALLA"),
		Port:     os.Getenv("POSTGRES_PORT_WALLA"),
	}

	checkDbCounts("HIREABLE", channelID, hireableConfig)
	checkDbCounts("WALLA", channelID, wallaConfig)
}

func checkDbCounts(dbName string, channelID string, config DbConfig) {
	connStr := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
		config.User, config.Database, config.Password, config.Host, config.Port)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	error_message := ""
	generic_error_message := dbName + " CheckAvgDbCounts error:"

	// clickers
	query_clicks := `
		WITH tiempo_actual AS (
			SELECT 
				(CASE 
					WHEN date_part('minute', now() - interval '3 hour') < 30 THEN date_trunc('hour', now() - interval '3 hour')
					ELSE date_trunc('hour', now() - interval '3 hour') + interval '1 hour'
				END)::time AS hora_fin,
				(CASE 
					WHEN date_part('minute', now() - interval '4 hour') < 30 THEN date_trunc('hour', now() - interval '4 hour')
					ELSE date_trunc('hour', now() - interval '4 hour') + interval '1 hour'
				END)::time AS hora_inicio
		),
		clicks_ultimo_dia AS (
			SELECT COUNT(*) AS day_before_clicks
			FROM clickers, tiempo_actual
			WHERE ts::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
			AND DATE(ts) = current_date - interval '1 day'
		),
		clicks_hoy AS (
			SELECT COUNT(*) AS clicks_hoy
			FROM clickers, tiempo_actual
			WHERE ts::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
			AND DATE(ts) = current_date
		)
		SELECT 
			hora_inicio,
			hora_fin,
			clicks_hoy.clicks_hoy,
			clicks_ultimo_dia.day_before_clicks,
			(clicks_hoy.clicks_hoy::float / clicks_ultimo_dia.day_before_clicks::float * 100) AS porcentaje_variacion
		FROM clicks_hoy, clicks_ultimo_dia, tiempo_actual;
		`

	var hora_inicio_clicks, hora_fin_clicks time.Time
	var clicks_hoy, day_before_clicks int
	var porcentaje_variacion_clicks float64
	var porcentaje_limite_aceptable_clicks float64 = 30.0

	err = db.QueryRow(query_clicks).Scan(&hora_inicio_clicks, &hora_fin_clicks, &clicks_hoy, &day_before_clicks,&porcentaje_variacion_clicks)
	if err != nil {
		log.Fatal("Failed to execute query: ", err)
	}

	hora_inicio_clicks_format := hora_inicio_clicks.Add(-3 * time.Hour).Format("15:04")
	hora_fin_clicks_format := hora_fin_clicks.Add(-3 * time.Hour).Format("15:04")

	error_message_clicks := fmt.Sprintf("El porcentaje de variación de clicks entre %s y %s = %.2f%%. \nClick de hoy: %d | Clicks de ayer: %d", hora_inicio_clicks_format, hora_fin_clicks_format, porcentaje_variacion_clicks, clicks_hoy, day_before_clicks)

	if porcentaje_variacion_clicks < porcentaje_limite_aceptable_clicks {
		error_message += error_message_clicks + "\n \n"
	}

	// api_job_search
	query_searchs := `
		WITH tiempo_actual AS (
			SELECT 
				(CASE 
					WHEN date_part('minute', now() - interval '3 hour') < 30 THEN date_trunc('hour', now() - interval '3 hour')
					ELSE date_trunc('hour', now() - interval '3 hour') + interval '1 hour'
				END)::time AS hora_fin,
				(CASE 
					WHEN date_part('minute', now() - interval '4 hour') < 30 THEN date_trunc('hour', now() - interval '4 hour')
					ELSE date_trunc('hour', now() - interval '4 hour') + interval '1 hour'
				END)::time AS hora_inicio
		),
		busquedas_ultimo_dia AS (
			SELECT COUNT(*) AS day_before_busquedas
			FROM api_job_searchs, tiempo_actual
			WHERE created_at::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
			AND DATE(created_at) = current_date - interval '1 day'
		),
		busquedas_hoy AS (
			SELECT COUNT(*) AS busquedas_hoy
			FROM api_job_searchs, tiempo_actual
			WHERE created_at::time BETWEEN tiempo_actual.hora_inicio AND tiempo_actual.hora_fin
			AND DATE(created_at) = current_date
		)
		SELECT 
			hora_inicio::time(0),
			hora_fin::time(0),
			busquedas_hoy.busquedas_hoy,
			busquedas_ultimo_dia.day_before_busquedas,
			(busquedas_hoy.busquedas_hoy::float / busquedas_ultimo_dia.day_before_busquedas::float * 100) AS porcentaje_variacion
		FROM busquedas_hoy, busquedas_ultimo_dia, tiempo_actual;
		`
	
	var hora_inicio_searchs, hora_fin_searchs time.Time
	var busquedas_hoy, day_before_busquedas int
	var porcentaje_variacion_searchs float64
	var porcentaje_limite_aceptable_searchs float64 = 30.0

	err = db.QueryRow(query_searchs).Scan(&hora_inicio_searchs, &hora_fin_searchs, &busquedas_hoy, &day_before_busquedas, &porcentaje_variacion_searchs)
	if err != nil {
		log.Fatal("Failed to execute query: ", err)
	}

	hora_inicio_searchs_format := hora_inicio_searchs.Add(-3 * time.Hour).Format("15:04")
	hora_fin_searchs_format := hora_fin_searchs.Add(-3 * time.Hour).Format("15:04")

	error_message_searchs := fmt.Sprintf("El porcentaje de variación de searchs entre %s y %s = %.2f%%. \nSearchs de hoy: %d | Searchs de ayer: %d", hora_inicio_searchs_format, hora_fin_searchs_format, porcentaje_variacion_searchs, busquedas_hoy, day_before_busquedas)

	if porcentaje_variacion_searchs < porcentaje_limite_aceptable_searchs {
		error_message += error_message_searchs
	}
	
	if(error_message != "") {
		slack_utils.SendMessage(generic_error_message + "\n" + error_message, channelID)
	} 
}
