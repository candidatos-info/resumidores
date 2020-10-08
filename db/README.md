# Tutorial

Para ativar esse resumidor é necessário que já tenham sido feitos o enriquecimento dos arquivos de candidaturas e de fotos. Uma vez que os mesmos estejam feitos, rode o seguinte comando:

```
go run *.go -candidaturesPaths=${CANDIDATURES_PATHS} -picturesReferences=${PICTURES_REFERENCES} -state=${STATE} -credentials=${CREDENTIALS} -OAuthToken=${OAUTH_TOKEN} -dbName=${DB_NAME} -dbURL=${DB_URL}
```

onde:
+ CANDIDATURES_PATHS é o arquivo contendo os paths dos arquivos de candidaturas locais e no Google Drive;
+ PICTURES_REFERENCES é o arquivo contento referência dos arquivos de fotos processados;
+ STATE é o estado a ser processado;
+ CREDENTIALS é o path para o arquivo de credenciais do Google Drive;
+ OAUTH_TOKEN é o path para o arquivo OAuth de acesso ao Google Drive;
+ DB_NAME é o nome do banco de dados;
+ DB_URL é a URL de conexão com o banco;

Um exemplo concreto:

```
go run *.go -candidaturesPaths=/Users/user0/candidatos.info/enri/candidatures/candidatures_path-2016-AL.csv -picturesReferences=/Users/user0/candidatos.info/enri/ballot_picture/handled_pictures-2016-AL -state=AL -credentials=/Users/user0/candidatos.info/enriquecedores/ballot_picture/credentials.json -OAuthToken=/Users/user0/candidatos.info/enriquecedores/ballot_picture/token.json -dbName=candidatos -dbURL=mongodb://localhost:27017/candidatos
```
