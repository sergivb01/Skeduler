# Notes

Desacoplar backend i els workers

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

## Pendent

- Server:
    - **HTTP**: reestructuració, posar tokens de seguretat, control exhaustiu d'errors
    - **Configuració**: afegir més opcions, variables d'entorn
    - **Main**: possibilitat de canviar la base de dades desitjada, control d'errors, flags, ...

- Worker:
    - **HTTP**: control exhaustiu d'errors, ...
    - **Configuració**: falta fer tot, idem servidor.
    - **Main**: idem
    - **Worker**: s'ha de fer un streaming de logs cap al servidor http. Reordenar?

- Streaming de logs: es pot fer amb sockets, http push (http 2.0), vàries requests (**no recomenable**), ...
- Compartir request bodies pels clients http -> podriem fer
  servir [protocol buffers](https://developers.google.com/protocol-buffers).
    - https://grpc.io/
    - https://medium.com/safetycultureengineering/grpc-over-http-3-53f41fc0761e
    - https://github.com/grpc/grpc-web
