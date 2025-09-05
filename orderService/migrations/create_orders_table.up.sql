create table orders (
                          "id"                serial        primary key,
                          "created_at"        timestamptz   not null    default now(),
                          "updated_at"        timestamptz   not null    default now(),
                          "number"            text          unique not null,
                          "customer_name"     text          not null,
                          "type"              text          not null check (type in ('dine_in', 'takeout', 'delivery')),
                          "table_number"      integer,
                          "delivery_address"  text,
                          "total_amount"      decimal(10,2) not null,
                          "priority"          integer       default 1,
                          "status"            text          default 'received',
                          "processed_by"      text,
                          "completed_at"      timestamptz
);