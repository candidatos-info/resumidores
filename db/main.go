package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/candidatos-info/descritor"
	"github.com/golang/protobuf/proto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	candidaturesCollection = "candidatures"
)

var (
	re = regexp.MustCompile(`([0-9]+)`) // regexp to extract numbers
)

// db schema
type votingCity struct {
	City       string
	State      string
	Candidates []*descritor.Candidatura
}

type gDriveCandFiles struct {
	candidatureFile *drive.File
	picture         *drive.File
}

func main() {
	source := flag.String("source", "", "local onde os arquivos de fotos e candidaturas estão aramazenados")
	state := flag.String("state", "", "estado para ser enriquecido")
	projectID := flag.String("projectID", "", "id do projeto no Google Cloud")
	googleDriveCredentialsFile := flag.String("credentials", "", "chave de credenciais o Goodle Drive")
	goodleDriveOAuthTokenFile := flag.String("OAuthToken", "", "arquivo com token oauth")
	flag.Parse()
	if *source == "" {
		log.Fatal("informe o local onde os arquivos de fotos e candidaturas estão")
	}
	if *state == "" {
		log.Fatal("informe o estado a ser processado")
	}
	if *projectID == "" {
		log.Fatal("informe o ID do projeto no GCP")
	}
	if *googleDriveCredentialsFile == "" {
		log.Fatal("informe o path para o arquivo de credenciais do Goodle Drive")
	}
	if *goodleDriveOAuthTokenFile == "" {
		log.Fatal("informe o path para o arquivo de token OAuth do Google Drive")
	}
	// Creating datastore client
	datastoreClient, err := datastore.NewClient(context.Background(), *projectID)
	if err != nil {
		log.Fatalf("falha ao criar cliente de datastore: %v", err)
	}
	// Creating Google Drive client
	googleDriveService, err := createGoogleDriveClient(*googleDriveCredentialsFile, *goodleDriveOAuthTokenFile)
	if err != nil {
		log.Fatalf("falha ao criar cliente do Google Drive, erro %q", err)
	}
	if err := summarize(*source, *state, datastoreClient, googleDriveService); err != nil {
		log.Fatalf("falha ao executar processamento do resumidor do banco, erro %q", err)
	}
}

func createGoogleDriveClient(googleDriveCredentialsFile, goodleDriveOAuthTokenFile string) (*drive.Service, error) {
	b, err := ioutil.ReadFile(googleDriveCredentialsFile)
	if err != nil {
		log.Fatalf("falha ao ler arquivo de crendenciais [%s], erro %q", googleDriveCredentialsFile, err)
	}
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("falha ao processar configuraçōes usando o arquivo [%s], erro %q", googleDriveCredentialsFile, err)
	}
	f, err := os.Open(goodleDriveOAuthTokenFile)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo de token oauth [%s], erro %q", goodleDriveOAuthTokenFile, err)
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err = json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("falha ao fazer bind do token OAuth, erro %q", err)
	}
	googleDriveService, err := drive.New(config.Client(context.Background(), tok))
	if err != nil {
		return nil, fmt.Errorf("não foi possível criar Google Drive service, erro %q", err)
	}
	return googleDriveService, nil
}

func summarize(source, state string, datastoreClient *datastore.Client, googleDriveService *drive.Service) error {
	query := fmt.Sprintf("name contains '%s' and '%s' in parents", state, source) // pegando os arquivos com prefixo 'estado' da pasta de id 'source'
	fileList, err := googleDriveService.Files.List().Q(query).Do()
	if err != nil {
		return fmt.Errorf("falha ao buscar arquivos do estado [%s] no diretório [%s], erro %q", state, source, err)
	}
	candFiles := getCandidateFiles(fileList)
	dbItems, err := getDBItems(candFiles, googleDriveService)
	if err != nil {
		return fmt.Errorf("falha ao gerar itens do banco, erro %q", err)
	}
	for _, c := range dbItems {
		userKey := datastore.NameKey(candidaturesCollection, fmt.Sprintf("%s_%s", c.State, c.City), nil)
		if _, err := datastoreClient.Put(context.Background(), userKey, c); err != nil {
			return fmt.Errorf("falha ao salvar cidade [%s] do estado [%s] no banco, erro %q", c.City, c.State, err)
		}
		log.Printf("saved city [%s] of state [%s]", c.City, c.State)
	}
	return nil
}

func getCandidateFiles(fileList *drive.FileList) map[string]gDriveCandFiles {
	candFiles := make(map[string]gDriveCandFiles)
	for _, item := range fileList.Files {
		sequencialID := re.FindAllString(item.Name, -1)[0]
		switch filepath.Ext(item.Name) {
		case ".pb":
			candFiles[sequencialID].picture = item // ISSUE CODE!
		case ".jpg":
			candFiles[sequencialID].picture = item // ISSUE CODE!
		default:
			log.Printf("file [%s] has unknown extension\n", item.Name)
		}
	}
	return candFiles
}

func getDBItems(candFiles map[string]gDriveCandFiles, googleDriveService *drive.Service) (map[string]*votingCity, error) {
	dbItems := make(map[string]*votingCity)
	for _, c := range candFiles {
		content, err := func() ([]byte, error) {
			response, err := googleDriveService.Files.Get(c.candidatureFile.Id).Download()
			if err != nil {
				return nil, fmt.Errorf("falha ao pegar bytes de arquivo de candidatura, erro %q", err)
			}
			defer response.Body.Close()
			b, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, fmt.Errorf("falha ao ler bytes de arquivo de candidatura, erro %q", err)
			}
			return b, nil
		}()
		if err != nil {
			return nil, err
		}
		time.Sleep(time.Second * 1) // esse delay é colocado para evitar atingir o limite de requests por segundo. Preste atenção ao tamanho do arquivo que irá enviar.
		var candidature descritor.Candidatura
		if err = proto.Unmarshal(content, &candidature); err != nil {
			return nil, fmt.Errorf("falha ao deserializar bytes de arquivo de candidatura para struct descritor.Candidatura, erro %q", err)
		}
		//TODO add Picture URL to struct
		if dbItems[candidature.Municipio] == nil {
			dbItems[candidature.Municipio] = &votingCity{
				City:       candidature.Municipio,
				State:      candidature.UF,
				Candidates: []*descritor.Candidatura{&candidature},
			}
		} else {
			dbItems[candidature.Municipio].Candidates = append(dbItems[candidature.Municipio].Candidates, &candidature)
		}
	}
	return dbItems, nil
}