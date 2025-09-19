### Wheres My Pizza
## Project Description

This project simulates a restaurant order system using Go, RabbitMQ, and PostgreSQL.
It shows how different services can work together like in a real pizza delivery app.

## The system has 4 services:

Order Service – takes customer orders through an HTTP API, saves them in the database, and sends them to RabbitMQ.

Kitchen Worker – receives orders from the queue, simulates cooking, updates order status, and sends notifications.

Tracking Service – lets users check order status, order history, and kitchen worker status through HTTP API.

Notification Service – listens for status updates and prints them (like customer notifications).

## How It Works

A customer creates an order using the Order Service.

The order is saved in PostgreSQL and sent to RabbitMQ.

A Kitchen Worker takes the order, marks it as cooking, simulates preparation, then marks it as ready.

The Notification Service shows updates when the order status changes.

The Tracking Service allows checking the current status and history of orders.

## Services
# Order Service

Endpoint: POST /orders

Saves order to database.

Publishes message to RabbitMQ.

1. Example request:

{
  "customer_name": "John Doe",
  "order_type": "takeout",
  "items": [
    { "name": "Margherita Pizza", "quantity": 1, "price": 15.99 },
    { "name": "Caesar Salad", "quantity": 1, "price": 8.99 }
  ]
}

Example response:

{
  "order_number": "ORD_20240919_001",
  "status": "received",
  "total_amount": 10.00
}

2. 
{
  "customer_name": "John",
  "order_type": "dine_in",
  "table_number": 2,
  "items": [
    { "name": "Margherita Pizza", "quantity": 1, "price": 15.99 },
    { "name": "Caesar Salad", "quantity": 1, "price": 8.99 }
  ]
}

3. 
{
  "customer_name": "John",
  "order_type": "delivery",
  "delivery_address": "addressssssssss",
  "items": [
    { "name": "Margherita Pizza", "quantity": 1, "price": 15.99 },
    { "name": "Caesar Salad", "quantity": 1, "price": 8.99 }
  ]
}


## Kitchen Worker

Registers itself in database.

Takes orders from RabbitMQ.

Updates order status (cooking → ready).

Sends status updates to RabbitMQ.

## Tracking Service

API endpoints:

GET /orders/{order_number}/status – check current status.

GET /orders/{order_number}/history – see order history.

GET /workers/status – list of kitchen workers.

## Notification Service

Listens for order updates.

Prints messages like:

Notification: Order ORD_20240919_001 changed from received to cooking by chef_mario

## Run Example

# Start each service with flags:

./restaurant-system --mode=order-service --port=3000
./restaurant-system --mode=kitchen-worker --worker-name="chef_anna" --order-types="takeout" --prefetch=1
./restaurant-system --mode=tracking-service --port=3002
./restaurant-system --mode=notification-subscriber


# Place an order:

curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "Jane Doe",
        "order_type": "takeout",
        "items": [
          {"name": "Margherita Pizza", "quantity": 1, "price": 15.99},
          {"name": "Caesar Salad", "quantity": 1, "price": 8.99}
        ]
      }'

# Place an tracker:
curl http://localhost:3002/orders/{order number}/status
curl http://localhost:3002/workers/status
