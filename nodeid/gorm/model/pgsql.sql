-- auto-generated definition
create table snowflake_kv
(
    key     text                     not null
        primary key,
    node_id bigserial,
    time    bigint                   not null,
    created timestamp with time zone not null,
    updated timestamp with time zone not null
);

comment on column snowflake_kv.key is 'Key';

comment on column snowflake_kv.node_id is 'Node ID';

comment on column snowflake_kv.time is 'time';

comment on column snowflake_kv.created is '创建时间';

comment on column snowflake_kv.updated is '更新时间';

alter table snowflake_kv
    owner to system;

create unique index "snowflake_kv_UN_node_id"
    on snowflake_kv (node_id);

