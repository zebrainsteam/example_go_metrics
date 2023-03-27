package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Shopify/sarama"
	"github.com/kybuk_oo/example_go_metrics/orders/pkg/broker"
	"github.com/kybuk_oo/example_go_metrics/orders/pkg/datastore"
	"github.com/kybuk_oo/example_go_metrics/orders/pkg/monitoring"
	"github.com/kybuk_oo/example_go_metrics/orders/transport"
	"github.com/rs/zerolog/log"
)

func main() {
	db := datastore.InitDB()
	producer := broker.InitKafkaProducer()

	handlers := map[string]sarama.ConsumerGroupHandler{
		os.Getenv("GOODS_CREATED_TOPIC"):  broker.BuildGoodsCreatedHandler(db),
		os.Getenv("GOODS_REJECTED_TOPIC"): broker.BuildGoodsRejectedHandler(db),
	}
	broker.RunConsumers(context.Background(), handlers)

	metrics, err := monitoring.StartMetrics()
	if err != nil {
		log.Error().Err(err).Msg("Server mertrics hasn't been started.")
		os.Exit(1)
	}

	fmt.Println("server metrics is starting...")

	fmt.Println("server is starting...")
	server := transport.NewServer(db, producer, metrics)
	err = server.Start()
	if err != nil {
		log.Error().Err(err).Msg("Server hasn't been started.")
		os.Exit(1)
	}
}
