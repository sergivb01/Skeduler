# Notes

Desacoblar backend i els workers
- [X] Worker:
    - [X] Configuració
    - [X] Client HTTP:
        - [X] polling per noves tasques
        - [X] push de noves línies de log
        - [X] push job finalitzat: ha de fer push de tot el JOB complet perquè l'estat i altres dades poden haver estat
          modificades

- [ ] Backend:
    - [ ] Configuració
    - [ ] Autenticació (tokens workers != tokens clients)
    - [ ] Endpoints:
        - [X] Crear nou job
        - [X] Obtenir job per id
        - [ ] Obtenir tots els jobs
        - [X] Obtenir tots els logs
        - [X] Streaming de logs

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

## Pendent

- Server:
    - **HTTP**: reestructuració, posar tokens de seguretat, control exhaustiu d'errors
    - **Configuració**: variables d'entorn?
    - **Main**: possibilitat de canviar la base de dades desitjada, control d'errors, flags, ...
    - **Bases de dades**: actualitzar sqlite

- Worker:
    - **HTTP**: control exhaustiu d'errors, ...
    - **Configuració**: variables d'entorn?
    - **Main**: idem
