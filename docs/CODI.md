# Codi

Està separat en les següents carpetes:

- cmd: conté els diferents mains per als executables.
    - server: servidor http i notificacions telegram
    - skeduler: client línia de comandes per encuar/consultar experiments
    - worker: fa la feina bruta
- internal
    - config: utilitat per llegir configuracions
    - database: conté les diferents implementacions de la base de dades
    - jobs: especificació de l'estructura d'un experiment

> Els diferents endpoints per la API REST estan especificats a la secció de servidor

La comunicació entre client <-> backend es fa a través de HTTP. Si es vol consultar
els logs en temps real, es pot fer per HTTP (
amb [Chunked transfer encoding](https://en.wikipedia.org/wiki/Chunked_transfer_encoding))
o mitjançant websockets (amb el query parameter `?ws`).

Per comunicar-se entre worker <-> backend, també es fa per HTTP. L'streaming de pujar
els logs es fa per websockets.

Hi ha un fitxer `setup.sql` amb la creació de la taula, els índexs i el tipus `job_status`. Adicionalment, hi ha
comentada unes línies per canviar el propietari de la taula i el tipus.