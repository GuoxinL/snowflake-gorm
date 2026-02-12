-- auto-generated definition
create table snowflake_kv
(
    `key`   varchar(191) not null comment 'Key'
        primary key,
    node_id bigint       not null comment 'Node ID',
    time    bigint       not null comment 'time',
    created datetime(3)  not null comment '创建时间',
    updated datetime(3)  not null comment '更新时间',
    constraint snowflake_kv_UN_node_id
        unique (node_id)
);

