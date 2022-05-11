# Client

- **All:** mostra la llista d'experiments. No espera cap paràmetre
- **Show:** mostra un experiment en concret. S'espera la ID (uuid).
- **Enqueue:** encua un experiment. S'espera la ruta a un fitxer **json** amb les especificacions.
- **Update:** actualitza la informació. S'espera la ruta a un fitxer **json** amb els canvis.
- **Logs:** donada una ID (uuid), mostra els logs de l'experiment fins a la data. Si s'utilitza la flag `-f`, se
  segueixen en temps real.
- **Help:** mostra el menú d'ajuda.

```
NAME:
   skeduler - Encuador d'experiments amb Docker

USAGE:
   skeduler [global options] command [command options] [arguments...]

AUTHORS:
   Sergi Vos <contacte@sergivos.dev>
   Xavier Terés <algo@xavierteres.com>

COMMANDS:
   all, a, ls  Lists all experiments
   show, s     Shows an experiments
   enqueue, e  Enqueues an experiment
   update, u   Updates an experiment
   logs, l     Shows an experiment's logs
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
```

## Configuració

La configuració es busca a `$HOME/.skeduler.json`. Els continguts són:

```json
{
  "host": "http://localhost:8080",
  "token": "token_autenticació"
}
```