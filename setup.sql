CREATE TYPE job_status AS ENUM ('ENQUEUED', 'RUNNING', 'FINISHED', 'CANCELLED');

CREATE TABLE jobs
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

CREATE INDEX jobs_status_index ON jobs (status);
CREATE INDEX jobs_createdat_index ON jobs (created_at);

ALTER TABLE jobs
    OWNER TO skeduler;
ALTER TYPE job_status OWNER TO skeduler;