# Skeduler

Encuador d'experiments.

[Documentació](./docs/README.md)

## Dades

- Tasques:
    - ID
    - Nom
    - Descripció
    - Prioritat
    - Docker
        - Imatge
        - Comanda
        - _Autenticació?_: per fer pull imatge
        - Variables d'entorn (mapa kv)
    - Status: QUEUED, RUNNING, FINISHED, ERROR, CANCELLED
    - Metadades: json
    - CreatedAt
    - UpdatedAt
