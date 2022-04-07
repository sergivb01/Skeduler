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
