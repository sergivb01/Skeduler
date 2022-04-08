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
    - Resultat
        - ExecutedAt
        - FinishedAt
        - GPUs
        - Duration
        - ExitCode
        - Error (string)
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

- Streaming de logs: es pot fer amb sockets, http push (http 2.0), vàries requests (**no recomenable**), ...
- Compartir request bodies pels clients http -> podriem fer
  servir [protocol buffers](https://developers.google.com/protocol-buffers).
    - https://grpc.io/
    - https://medium.com/safetycultureengineering/grpc-over-http-3-53f41fc0761e
    - https://github.com/grpc/grpc-web
    - https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
    - https://grpc.io/docs/guides/auth/
    - https://grpc.io/docs/what-is-grpc/introduction/
    - https://levelup.gitconnected.com/grpc-how-to-make-client-streaming-calls-5c731197585
    - https://grpc.io/docs/what-is-grpc/core-concepts/
    - https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
    - https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
    - https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md
    - https://github.com/grpc/grpc-go/blob/master/examples/features/authentication/server/main.go
    - https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto
    - https://medium.com/utility-warehouse-technology/advanced-grpc-error-usage-1b37398f0ff4
    - https://cloud.google.com/apis/design/errors
    - https://www.reddit.com/r/golang/comments/epfryk/go_grpc_server_project_structure/
    - https://sahansera.dev/building-grpc-server-go/
    - https://medium.com/@nate510/structuring-go-grpc-microservices-dd176fdf28d0
