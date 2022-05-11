# Server

Totes les peticions retornen un status "200 OK" si tot és correcte.

Exemple de resposta d'un experiment:

```json
{
  "id": "94f1bd4a-e989-402f-a96e-d2c1dda46e22",
  "name": "aaaaaaaaaa",
  "description": "test example desc",
  "docker": {
    "image": "nvidia/cuda:11.0-base",
    "command": "nvidia-smi",
    "environment": {
      "TESTING": 123456789
    }
  },
  "created_at": "2022-05-11T09:31:10.38384Z",
  "updated_at": "2022-05-11T09:31:10.38384Z",
  "status": "ENQUEUED",
  "metadata": [
    1,
    2,
    3,
    {
      "test": "yes"
    }
  ]
}
```

Job status és un enum de:

- `ENQUEUED`
- `RUNNING`
- `FINISHED`
- `CANCELLED`

### GET /experiments

Retorna la llista d'experiments

### POST /experiments

El cos de la petició ha de ser un job sense status, created_at, updated_at, status, id. Retorna l'experiment complet.

### GET /experiments/{id}

Retorna un experiment o status code "404 Not Found" si no existeix.

### PUT /experiments/{id}

Actualitza l'experiment. Cos:

```json
{
  "name": "nom",
  "description": "descripció",
  "metadata": {},
  "status": "ENQUEUED"
}
```

### GET /logs/{id}

Retorna els logs en plaintext.

### GET /logs/{id}/tail

Retorna els logs en plaintext en temps real. Amb el query parameter `?ws` s'intentarà fer un upgrade de la connexió i
transmetre els logs per websocket. Els missatges seràn de tipus binari (`websocket.BinaryMessage`) i cada missatge una
línia de log.

## Configuració

```yaml
database: "postgres://user:password@host:5432/database_name"

telegram_token: "telegram_token"

http:
  listen: ":8080"
  read_timeout: "15s"
  idle_timeout: "15s"

tokens:
  - "token_1"
  - "token_2"
  - "token_3"
```

## Bases de dades

```sql
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

CREATE INDEX jobs_status_index ON jobs (status, created_at);

-- Opcionals:
-- ALTER TABLE jobs OWNER TO skeduler;
-- ALTER TYPE job_status OWNER TO skeduler;
```