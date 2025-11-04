Wheres My Pizza 🍕

A distributed restaurant order management system built with Go, RabbitMQ, and PostgreSQL, simulating a real-world pizza delivery workflow. The system demonstrates microservices architecture, message queues, and asynchronous service communication.


🛠️ Technologies Used

Backend: Go, RabbitMQ, PostgreSQL

Messaging: AMQP 0.91 (RabbitMQ)

Database Driver: pgx/v5

Deployment: Local / Docker

✨ Features

Four independent services: Order, Kitchen Worker, Tracking, Notification

Asynchronous communication using RabbitMQ

Real-time order tracking and notifications

Supports dine-in, takeout, and delivery orders

Structured JSON logging for all services

Graceful shutdown and error handling

📦 Installation

Clone the repository:

git clone git@github.com:AikaBe/-wheres-my-pizza-System.git
cd Wheres-My-Pizza


Build the project:

go build -o restaurant-system .


Make sure PostgreSQL and RabbitMQ are running.

Configure config.yaml with database and RabbitMQ connection info:

database:
host: localhost
port: 5432
user: restaurant_user
password: restaurant_pass
database: restaurant_db

rabbitmq:
host: localhost
port: 5672
user: guest
password: guest

🎯 Usage
Start each service
./restaurant-system --mode=order-service --port=3000
./restaurant-system --mode=kitchen-worker --worker-name="chef_anna" --order-types="takeout" --prefetch=1
./restaurant-system --mode=tracking-service --port=3002
./restaurant-system --mode=notification-subscriber

Place an order
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

Track order status
curl http://localhost:3002/orders/{order_number}/status
curl http://localhost:3002/workers/status

🏗️ System Architecture
+-------------------------+
|   PostgreSQL Database   |
|   (Order Storage)       |
+-----+-------------------+
^        ^
(Read/Write)   |        | (Read/Write)
v        v
+-----------+          +-----------+      +----------------+
| HTTP Client|-------->| Order     |      | Kitchen Worker |
| (curl etc) |          | Service   |      | Service       |
+-----------+          +-----------+      +----------------+
|                     |                 ^
|                     v                 |
|             +----------------+        |
|             | RabbitMQ Broker|<-------+
|             +----------------+
|                     |
v                     v
+----------------+     +-----------------+
| Notification   |     | Tracking Service|
| Subscriber     |     +-----------------+
+----------------+

✨ Services
1️⃣ Order Service

Receives orders via HTTP API

Validates input and saves orders to PostgreSQL

Publishes orders to RabbitMQ

Endpoints:

POST /orders – create new order

Order Example:

{
"customer_name": "John Doe",
"order_type": "takeout",
"items": [
{"name": "Margherita Pizza", "quantity": 1, "price": 15.99}
]
}

2️⃣ Kitchen Worker

Consumes orders from RabbitMQ

Updates order status (cooking → ready)

Publishes status updates to notifications

Flags:

--worker-name – unique worker name

--order-types – comma-separated order types

--prefetch – limit of unacknowledged messages

3️⃣ Tracking Service

Read-only HTTP API

Queries PostgreSQL to return:

Current order status

Order history

Kitchen worker status

Endpoints:

GET /orders/{order_number}/status

GET /orders/{order_number}/history

GET /workers/status

4️⃣ Notification Service

Subscribes to status updates from RabbitMQ

Prints human-readable notifications

Example Output:

Notification: Order ORD_20241216_001 changed from received to cooking by chef_mario

🔮 Learning Objectives

Microservices architecture

Message Queue systems with RabbitMQ

Asynchronous service communication

PostgreSQL database integration

Structured JSON logging

Graceful shutdown and error handling

🔧 Future Improvements

Web-based frontend for customers and kitchen staff

Push notifications via email/SMS

Docker Compose setup for easier deployment

Metrics and monitoring dashboards