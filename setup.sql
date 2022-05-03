create type job_status as enum ('ENQUEUED', 'RUNNING', 'FINISHED', 'CANCELLED');

create table if not exists jobs
(
    id                 uuid                     default gen_random_uuid()      not null
        primary key,
    name               text                                                    not null,
    description        text                                                    not null,
    docker_image       text                                                    not null,
    docker_command     text                                                    not null,
    docker_environment jsonb                                                   not null,
    created_at         timestamp with time zone default CURRENT_TIMESTAMP      not null,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP      not null,
    status             job_status               default 'ENQUEUED'::job_status not null,
    metadata           json
);



