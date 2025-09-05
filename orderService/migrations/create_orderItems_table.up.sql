create table order_items (
                             "id"          serial        primary key,
                             "created_at"  timestamptz   not null    default now(),
                             "order_id"    integer       references orders(id),
                             "name"        text          not null,
                             "quantity"    integer       not null,
                             "price"       decimal(8,2)  not null
);