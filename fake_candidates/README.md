# Tutorial

## Para adicionar candidatos de teste
Para adicionar candidatos de teste no sistema forneça um arquivo JSON como o exemplo neste diretório contendo os dados do candidato. Uma vez de posse do arquivo rode o seguinte comando usando o CLI:
```
$ go run *.go -fakeCandidatesFilePath=${FAKE_CANDIDATES_FILE} -dbName=${DB_NAME} -dbURL=${DB_URL}
```

Onde:
+ FAKE_CANDIDATES_FILE é o path para o arquivo JSON contendo os dados de candidatos falsos;
+ DB_NAME é o nome do banco de dados;
+ DB_URL é a URL de conexão com o banco;

Um exemplo completo seria:
```
$ go run *.go -fakeCandidatesFilePath=fake_candidates.json -dbName=candidatos -dbURL=mongodb://localhost:27017/candidatos
```

OBS: Esse resumidor irá salvar o sequencial_candidate com o caractere '@' na frente. Isso ajudará o frontend a distinguir o candidato de teste dos oficiais.

## Para remover candidatos de teste
Para remover candidatos de teste use o seguinte comando:

```
$ go run *.go -emailToRemove=${EMAIL_TO_REMOVE} -dbName=${DB_NAME} -dbURL=${DB_URL} -year=${YEAR}
```

Onde:
+ EMAIL_TO_REMOVE é o email para ser removido.
+ DB_NAME é o nome do banco de dados;
+ DB_URL é a URL de conexão com o banco;
+ YEAR ano da eleição para remover;

Um exemplo concreto da execução desse comando é o seguinte:

```
go run *.go -emailToRemove=danielfireman@gmail.com -dbName=candidatos -dbURL=mongodb://localhost:27017/candidatos -yea
r=2020
```
