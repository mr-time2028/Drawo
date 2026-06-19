create extension if not exists "uuid-ossp";

create table users
(
    id           uuid         not null primary key default uuid_generate_v4(),
    username     varchar(100) not null unique,
    password     varchar(60)  not null,
    is_active    boolean      not null             default false,
    is_superuser boolean      not null             default false,
    created_at   bigint       not null             default (extract(epoch from now())),
    updated_at   bigint,
    deleted_at   bigint
);
