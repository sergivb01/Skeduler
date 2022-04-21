# Notes

Desacoblar backend i els workers

- Worker:
    - Configuració
    - Client HTTP:
        - polling per noves tasques
        - push de noves línies de log
        - push job finalitzat: ha de fer push de tot el JOB complet perquè l'estat i altres dades poden haver estat
          modificades

- Backend:
    - Configuració
    - Endpoints:
        - Crear nou job
        - Obtenir job per id
        - Obtenir tots els logs
        - Streaming de logs

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
    - **Configuració**: afegir més opcions, variables d'entorn
    - **Main**: possibilitat de canviar la base de dades desitjada, control d'errors, flags, ...
    - **Bases de dades**: centrar-se només en PostgreSQL i deixar d'utilitzar el "RETURNING".

- Worker:
    - **HTTP**: control exhaustiu d'errors, ...
    - **Configuració**: falta fer tot, idem servidor.
    - **Main**: idem
    - **Worker**: s'ha de fer un streaming de logs cap al servidor http. Reordenar?
    - **Logger**: per cada tasca fer un logger que escriu al buffer (amb el que farem _streaming_ cap al servidor) i a
      stderr amb camps que identifiquen el contenidor i la ID tasca
