CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table rooms
(
    id    uuid         not null primary key default uuid_generate_v4(),
    title varchar(100) not null
);
