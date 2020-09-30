# Tutorial

Para executar o detector de candidatura recorrentes execute os seguintes comandos:
```
$ go build
$ ./reccurence_detector -dbName=${DB_NAME} -dbURL=${DB_URL} -currentElectionYear=${CURRENT_ELECTION_YEAR} -prevElectionYear=${PREV_ELECTION_YEAR} -state=${AL} -offset=${OFFSET}
```

Onde:
+ DB_NAME é o nome do banco;
+ DB_URL é URL de conexão com o banco;
+ CURRENT_ELECTION_YEAR é o ano da eleição atual;
+ PREV_ELECTION_YEAR é o ano da eleição que se deseja fazer a comparação;

```
$ ./reccurence_detector -dbName=candidatos -dbURL=mongodb://localhost:27017/candidatos -currentElectionYear=2020 -prevElectionYear=2016 -state=AL -offset=1
```
