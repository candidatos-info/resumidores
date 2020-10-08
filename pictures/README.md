# Tutorial

Para executar esse resumidor é necessário que o enriquecedor de fotos já tenha sido executado pois precisamos do arquivo contendo as referências das fotos. Uma vez feito isso, execute o seguinte comando:

```
go run *.go -dbName=${DB_NAME} -dbURL=${DB_URL} -picturesReferences=${PICTURES_REFERENCE} -year=${YEAR} -offset=${OFFSET}
```
Onde:
+ DB_NAME é o nome do banco de dados;
+ DB_URL é a URL de conexão com o banco;
+ PICTURES_REFERENCE é o path para o arquivo contendo as referências das fotos;
+ YEAR é o ano a ser processado;
+ OFFSET é o ponto de início do processamento. Por padrão ele é 0;

Um exemplo concreto:

```
go run *.go -dbName=candidatos -dbURL=mongodb://localhost:27017/candidatos -picturesReferences=/Users/user0/candidatos.i
nfo/enri/ballot_picture/pictures_references-2020-AL.csv -year=2020 -offset=0
```