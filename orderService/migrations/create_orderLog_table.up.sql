create table order_status_log (
                                  "id"          serial        primary key,
                                  "created_at"  timestamptz   not null    default now(),
                                  "order_id"    integer       references orders(id),
                                  "status"      text,
                                  "changed_by"  text,
                                  "changed_at"  timestamptz   default current_timestamp,
                                  "notes"       text
);