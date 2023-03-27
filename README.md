## Пример использования метрик в приложениях на GOLang ##
### Проект основан на паттерне микросервисной архитектуры SAGA ###
Брокер сообщений KAFKA

Запуск `docker-compose up -d`

Создание заказа
`curl --request POST \
   --header "Content-Type: application/json" \
   --data '{"user_id":1,"goods_ids":[1,2]}' \
   'http://localhost:8080/v1/orders'`
либо можно использовать коллекцию для Postman (в корне репозитория)

Визуализация мониторинга Prometheus+Grafana

Несколько готовых Dashboard для Grafana находятся в директории `grafana_dashboard` 