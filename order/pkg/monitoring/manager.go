package monitoring

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

type Metrics struct {
	Counter   map[string]*prometheus.CounterVec
	Gauge     map[string]prometheus.Gauge
	Summary   map[string]prometheus.Summary
	Histogram map[string]*prometheus.HistogramVec
}

func StartMetrics() (Metrics, error) {
	counters := Metrics{
		Counter:   make(map[string]*prometheus.CounterVec),
		Gauge:     make(map[string]prometheus.Gauge),
		Summary:   make(map[string]prometheus.Summary),
		Histogram: make(map[string]*prometheus.HistogramVec),
	}
	/*
		Counter, как несложно угадать по названию, представляет собой простой счетчик.
		Не самый полезный тип метрики, поскольку счетчик этот является неубывающим.
		То есть, подходит он для отображения только чего-то вроде суммарного числа учетных записей в системе, да и то лишь при условии, что учетные записи являются неудаляемыми.
	*/
	/*
		# HELP request_send Количество вызовов запросов
		# TYPE request_send counter
		request_send{type="какой то тип запроса"} 5
	*/
	createOrderSend := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "example_go_metrics_orders",
		Name:      "request_send",
		Help:      "Количество вызовов запросов",
	}, []string{"type"})
	counters.Counter["request_send"] = createOrderSend

	/*
		Gauge, здесь используется в значении «мера».
		Gauge похож на Counter, но в отличие от него может не только возрастать, но и убывать.
		Этот тип отлично подходит для отображения текущего значения чего-то
		— температуры, давления, числа пользователей онлайн, и так далее.
	*/
	/*
		# HELP work_order_create Количество активных вызовов создания заказа
		# TYPE work_order_create gauge
		work_order_create 3
	*/
	workOrderCreate := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "example_go_metrics_orders",
			Name:      "work_order_create",
			Help:      "Количество активных вызовов создания заказа",
		})
	counters.Gauge["work_order_create"] = workOrderCreate
	/*
		Histogram представляет собой гистограмму.
		Этот тип метрики хранит число раз, которое измеряемая величина попала в заданный интервал значений (бакет).
		Гистограммы может быть трудновато использовать, если интервал допустимых значений величины заранее неизвестен.
		В Grafana по гистограмме можно примерно посчитать процентили, используя функцию histogram_quantile.
	*/
	/*
		# HELP request_processing_time_histogram_ms Продолжительность выполнения запроса
		# TYPE request_processing_time_histogram_ms histogram
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="0.1"} 0
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="0.15"} 0
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="0.2"} 0
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="0.25"} 0
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="0.3"} 0
		request_processing_time_histogram_ms_bucket{status="какой то статус", method="какой то метод", le="+Inf"} 2
		request_processing_time_histogram_ms_sum 3.5001457279999997
		request_processing_time_histogram_ms_count 2
	*/

	requestProcessingTimeHistogramMs := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "example_go_metrics_orders",
			Name:      "request_processing_time_histogram_ms",
			Help:      "Продолжительность исполнения запроса Histogram",
			// Можно указать свой тип бакета или оставить значение по умолчанию
			// Buckets: []float64{0.5, 10, 20},
			// Buckets: ExponentialBuckets(100, 1.2, 3),
			// Buckets: LinearBuckets(-15, 5, 6),
			Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		}, []string{"status", "method"})
	counters.Histogram["request_processing_time_histogram_ms"] = requestProcessingTimeHistogramMs
	/*
		Summary честно считает заданные процентили.
		Идеально подходит для измерения времени ответа или чего-то такого.
		Минус Summary заключается в том, что его дорого считать.
		Поэтому часто обходятся гистограммами и примерными значениями процентилей.
	*/
	/*
		# HELP request_processing_time_summary_ms Время задержки обработки запроса в секундах
		# TYPE request_processing_time_summary_ms summary
		request_processing_time_summary_ms{quantile="0.5"} 1.000199219
		request_processing_time_summary_ms{quantile="0.9"} 2.5
		request_processing_time_summary_ms{quantile="0.99"} 2.5
		request_processing_time_summary_ms_sum 3.5001992189999998
		request_processing_time_summary_ms_count 2
	*/
	requestProcessingTimeSummaryMs := prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: "example_go_metrics_orders",
			Name:      "request_processing_time_summary_ms",
			Help:      "Продолжительность исполнения запроса Summary",
			/* В Objectives определяют оценки квантильного ранга с их соответствующими
			абсолютными ошибками. Если Objectives[q] = e, то значение, сообщаемое для q
			будет значением квантиля φ для некоторого φ между q-e и q+e.
			Значение по умолчанию — пустая карта, что приводит к сводке без
			квантили.*/
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})
	counters.Summary["request_processing_time_summary_ms"] = requestProcessingTimeSummaryMs
	/*
		Типы HistogramVec, SummaryVec и так далее.
		Они представляют собой словарь (map) из описанных выше типов.
		То есть, это как бы метрики со строковыми метками, или создаваемые на лету метрики.
		Отлично подходят в случаях, когда вам нужно измерить время ответа сервера в зависимости от запроса, или вроде того.
	*/

	metricsProm, err := RunPrometheus(counters)
	if err != nil {
		return metricsProm, err
	}

	return metricsProm, nil
}

func RunPrometheus(metrics Metrics) (Metrics, error) {
	var err error

	fmt.Println("start server metrics...")

	if len(metrics.Counter) > 0 {
		for _, counter := range metrics.Counter {
			////	prometheus.MustRegister(counter)
			err = prometheus.Register(counter)
			if err != nil {
				return Metrics{}, err
			}
		}
	}

	if len(metrics.Gauge) > 0 {
		for _, gauge := range metrics.Gauge {
			////	prometheus.MustRegister(counter)
			err = prometheus.Register(gauge)
			if err != nil {
				return Metrics{}, err
			}
		}
	}

	if len(metrics.Summary) > 0 {
		for _, summary := range metrics.Summary {
			////	prometheus.MustRegister(counter)
			err = prometheus.Register(summary)
			if err != nil {
				return Metrics{}, err
			}
		}
	}

	if len(metrics.Histogram) > 0 {
		for _, histogram := range metrics.Histogram {
			////	prometheus.MustRegister(counter)
			err = prometheus.Register(histogram)
			if err != nil {
				return Metrics{}, err
			}
		}
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err = http.ListenAndServe(":"+os.Getenv("METRICS_PORT"), nil)
		log.Println(err)
	}()

	return metrics, nil
}
