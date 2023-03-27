package transport

import (
	"context"
	"encoding/json"
	"github.com/kybuk_oo/example_go_metrics/orders/pkg/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kybuk_oo/example_go_metrics/orders/pkg/model"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router        *mux.Router
	db            *pgxpool.Pool
	kafkaProducer sarama.SyncProducer
	metrics       monitoring.Metrics
}

func NewServer(db *pgxpool.Pool, kafkaProducer sarama.SyncProducer, metrics monitoring.Metrics) Server {
	s := Server{}
	s.kafkaProducer = kafkaProducer
	s.db = db
	s.metrics = metrics
	s.router = mux.NewRouter()

	s.router.HandleFunc("/v1/orders", s.CreateOrderV1).Methods(http.MethodPost)

	return s
}

func (s Server) Start() error {
	return http.ListenAndServe(":"+os.Getenv("HTTP_BIND"), s.router)
}

func (s Server) CreateOrderV1(w http.ResponseWriter, r *http.Request) {
	//### START Метрика количества активных вызовов создания заказа Increment
	s.metrics.Gauge["work_order_create"].Inc()
	//### END Метрика количества активных вызовов создания заказа Increment
	now := time.Now()
	sleep(200)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Data hasn't been read.")
		w.WriteHeader(http.StatusBadRequest)
		//### START Метрика продолжительности вызова создания заказа Histogram
		s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusBadRequest)}).Observe(time.Since(now).Seconds())
		//### END Метрика продолжительности вызова создания заказа Histogram
		//### START Метрика количества результатов создания заказа
		s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_failed_bad_request"}).Inc()
		//### END Метрика количества результатов создания заказа
		//### START Метрика количества активных вызовов создания заказа Decrement
		s.metrics.Gauge["work_order_create"].Dec()
		//### END Метрика количества активных вызовов создания заказа Decrement
		return
	}

	orderData := model.OrderData{}
	err = json.Unmarshal(body, &orderData)
	if err != nil {
		log.Error().Err(err).Msg("Data hasn't been parsed.")
		w.WriteHeader(http.StatusBadRequest)
		//### START Метрика продолжительности вызова создания заказа Histogram
		s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusBadRequest)}).Observe(time.Since(now).Seconds())
		//### END Метрика продолжительности вызова создания заказа Histogram
		//### START Метрика количества результатов создания заказа
		s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_failed_bad_request"}).Inc()
		//### END Метрика количества результатов создания заказа
		//### START Метрика количества активных вызовов создания заказа Decrement
		s.metrics.Gauge["work_order_create"].Dec()
		//### END Метрика количества активных вызовов создания заказа Decrement
		return
	}

	var orderID int64
	err = s.db.QueryRow(context.Background(), `INSERT INTO orders (user_id, status_id, created_at) VALUES ($1, 1, NOW()) RETURNING id`, orderData.UserID).Scan(&orderID)
	if err != nil {
		log.Error().Err(err).Msg("Order hasn't been created.")
		w.WriteHeader(http.StatusInternalServerError)
		//### START Метрика продолжительности вызова создания заказа Histogram
		s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusInternalServerError)}).Observe(time.Since(now).Seconds())
		//### END Метрика продолжительности вызова создания заказа Histogram
		//### START Метрика количества результатов создания заказа
		s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_failed_server"}).Inc()
		//### END Метрика количества результатов создания заказа
		//### START Метрика количества активных вызовов создания заказа Decrement
		s.metrics.Gauge["work_order_create"].Dec()
		//### END Метрика количества активных вызовов создания заказа Decrement
		return
	}

	msg := model.CreatedOrderMsg{Data: model.Order{
		ID:       orderID,
		GoodsIds: orderData.GoodsIds,
	}}
	msgStr, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("Message hasn't been marshaled.")
		w.WriteHeader(http.StatusInternalServerError)
		//### START Метрика продолжительности вызова создания заказа Histogram
		s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusInternalServerError)}).Observe(time.Since(now).Seconds())
		//### END Метрика продолжительности вызова создания заказа Histogram
		//### START Метрика количества результатов создания заказа
		s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_failed_server"}).Inc()
		//### END Метрика количества результатов создания заказа
		//### START Метрика количества активных вызовов создания заказа Decrement
		s.metrics.Gauge["work_order_create"].Dec()
		//### END Метрика количества активных вызовов создания заказа Decrement
		return
	}

	producerMsg := &sarama.ProducerMessage{Topic: os.Getenv("ORDER_CREATED_TOPIC"), Value: sarama.StringEncoder(msgStr)}
	_, _, err = s.kafkaProducer.SendMessage(producerMsg)
	if err != nil {
		log.Error().Err(err).Msg("Message hasn't been sent.")
		w.WriteHeader(http.StatusInternalServerError)
		//### START Метрика продолжительности вызова создания заказа Histogram
		s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusInternalServerError)}).Observe(time.Since(now).Seconds())
		//### END Метрика продолжительности вызова создания заказа Histogram
		//### START Метрика количества результатов создания заказа
		s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_failed_server"}).Inc()
		//### END Метрика количества результатов создания заказа
		//### START Метрика количества активных вызовов создания заказа Decrement
		s.metrics.Gauge["work_order_create"].Dec()
		//### END Метрика количества активных вызовов создания заказа Decrement
		return
	}

	//### START Метрика количества результатов создания заказа
	s.metrics.Counter["request_send"].With(prometheus.Labels{"type": "request_order_success"}).Inc()
	//### END Метрика количества результатов создания заказа
	//### START Метрика продолжительности вызова создания заказа Histogram
	s.metrics.Histogram["request_processing_time_histogram_ms"].With(prometheus.Labels{"method": "CreateOrderV1", "status": strconv.Itoa(http.StatusOK)}).Observe(time.Since(now).Seconds())
	//### END Метрика продолжительности вызова создания заказа Histogram
	//### START Метрика продолжительности вызова создания заказа Summary
	s.metrics.Summary["request_processing_time_summary_ms"].Observe(time.Since(now).Seconds())
	//### END Метрика продолжительности вызова создания заказа Summary
	//### START Метрика количества активных вызовов создания заказа Decrement
	s.metrics.Gauge["work_order_create"].Dec()
	//### END Метрика количества активных вызовов создания заказа Decrement
}

func sleep(ms int) {
	rand.Seed(time.Now().UnixNano())
	now := time.Now()
	n := rand.Intn(ms + now.Second())
	time.Sleep(time.Duration(n) * time.Millisecond)
}
