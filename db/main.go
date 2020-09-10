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
	statesCollection       = "states"
	pageSize               = 1000
)

var (
	re = regexp.MustCompile(`([0-9]+)`) // regexp to extract numbers
)

// struct with a fields portion of descritor.Candidature. This is struct
// is used only for DB purposes.
type candidateForDB struct {
	SequencialCandidate string `datastore:"sequencial_candidate,omitempty"` // Sequencial code of candidate on TSE system.
	Site                string `datastore:"site,omitempty"`                 // Site of candidate.
	Facebook            string `datastore:"facebook,omitempty"`             // Facebook of candidate.
	Twitter             string `datastore:"twitter,omitempty"`              // Twitter of candidate.
	Instagram           string `datastore:"instagram,omitempty"`            // Instagram of candidate.
	Description         string `datastore:"description,omitempty"`          // Description of candidate.
	Biography           string `datastore:"biography,omitempty"`            // Biography of candidate.
	PhotoURL            string `datastore:"photo_url,omitempty"`            // Photo URL of candidate.
	LegalCode           string `datastore:"legal_code,omitempty"`           // Brazilian Legal Code (CPF) of candidate.
	Party               string `datastore:"party,omitempty"`                // Party of candidate.
	Name                string `datastore:"name,omitempty"`                 // Natural name of candidate.
	BallotName          string `datastore:"ballot_name,omitempty"`          // Ballot name of candidate.
	BallotNumber        int    `datastore:"ballot_number,omitempty"`        // Ballot number of candidate.
	Email               string `datastore:"email,omitempty"`                // Email of candidate.
}

// db schema
type votingCity struct {
	Year       int               `datastore:"year,omitempty"`       // Election year.
	City       string            `datastore:"city,omitempty"`       // Election city.
	State      string            `datastore:"state,omitempty"`      // Election state
	Candidates []*candidateForDB `datastore:"candidates,omitempty"` // List contaning the candidates of city.
}

// used on states collection
type state struct {
	State  string   `datastore:"state"`  // State name.
	Cities []string `datastore:"cities"` // List contaning the cities of state.
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

func summarize(source, s string, datastoreClient *datastore.Client, googleDriveService *drive.Service) error {
	query := fmt.Sprintf("name contains '%s' and '%s' in parents", s, source) // pegando os arquivos com prefixo 'estado' da pasta de id 'source'
	var result *drive.FileList
	var setToResolve []*drive.File
	var err error
	for result == nil || result.NextPageToken != "" {
		listRequest := googleDriveService.Files.List().Q(query)
		listRequest.PageSize(pageSize)
		listRequest.Fields("nextPageToken, files(id, name)")
		if result != nil {
			listRequest.PageToken(result.NextPageToken)
		}
		result, err = listRequest.Do()
		if err != nil {
			return fmt.Errorf("falha ao buscar arquivos do estado [%s] no diretório [%s], erro %q", s, source, err)
		}
		setToResolve = append(setToResolve, result.Files...)
	}
	candFiles := getCandidateFiles(setToResolve)
	dbItems, err := getDBItems(candFiles, googleDriveService)
	if err != nil {
		return fmt.Errorf("falha ao gerar itens do banco, erro %q", err)
	}
	citiesMap := make(map[string]bool)
	for _, c := range dbItems {
		citiesMap[c.City] = true
		userKey := datastore.NameKey(candidaturesCollection, fmt.Sprintf("%s_%s", c.State, c.City), nil)
		if _, err := datastoreClient.Put(context.Background(), userKey, c); err != nil {
			return fmt.Errorf("falha ao salvar cidade [%s] do estado [%s] no banco, erro %q", c.City, c.State, err)
		}
		log.Printf("saved city [%s] of state [%s]\n", c.City, c.State)
	}
	var cities []string
	for key := range citiesMap {
		cities = append(cities, key)
	}
	stateToSave := &state{
		State:  s,
		Cities: cities,
	}
	stateKey := datastore.NameKey(statesCollection, s, nil)
	if _, err := datastoreClient.Put(context.Background(), stateKey, stateToSave); err != nil {
		return fmt.Errorf("falha ao salvar estado [%s] na coleção de estado, erro %q", s, err)
	}
	return nil
}

func getCandidateFiles(fileList []*drive.File) map[string]gDriveCandFiles {
	candFiles := make(map[string]gDriveCandFiles)
	for _, item := range fileList {
		sequencialID := re.FindAllString(item.Name, -1)[0]
		switch filepath.Ext(item.Name) {
		case ".pb":
			c, ok := candFiles[sequencialID]
			if !ok {
				candFiles[sequencialID] = gDriveCandFiles{
					candidatureFile: item,
				}
			} else {
				c.candidatureFile = item
				candFiles[sequencialID] = c
			}
		case ".jpg":
			c, ok := candFiles[sequencialID]
			if !ok {
				candFiles[sequencialID] = gDriveCandFiles{
					picture: item,
				}
			} else {
				c.picture = item
				candFiles[sequencialID] = c
			}
		default:
			log.Printf("file [%s] has unknown extension\n", item.Name)
		}
	}
	return candFiles
}

func getDBItems(candFiles map[string]gDriveCandFiles, googleDriveService *drive.Service) (map[string]*votingCity, error) {
	dbItems := make(map[string]*votingCity)
	for _, c := range candFiles {
		if c.candidatureFile != nil {
			content, err := func() ([]byte, error) {
				r, err := googleDriveService.Files.Get(c.candidatureFile.Id).Download()
				if err != nil {
					return nil, fmt.Errorf("falha ao pegar bytes de arquivo de candidatura, erro %q", err)
				}
				defer r.Body.Close()
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return nil, fmt.Errorf("falha ao ler bytes de arquivo de candidatura, erro %q", err)
				}
				return b, nil
			}()
			if err != nil {
				return nil, err
			}
			log.Printf("downloaded protocol buffer for file [%s]\n", c.candidatureFile.Name)
			time.Sleep(time.Second * 1) // esse delay é colocado para evitar atingir o limite de requests por segundo. Preste atenção ao tamanho do arquivo que irá enviar.
			var candidature descritor.Candidatura
			if err = proto.Unmarshal(content, &candidature); err != nil {
				return nil, fmt.Errorf("falha ao deserializar bytes de arquivo de candidatura para struct descritor.Candidatura, erro %q", err)
			}
			if c.picture != nil { // se candidato tiver foto
				candidature.Candidato.PhotoURL = fmt.Sprintf("https://drive.google.com/uc?id=%s&export=download", c.picture.Id)
			}
			candidateDataToPersist := candidateForDB{
				SequencialCandidate: candidature.SequencialCandidato,
				Site:                candidature.Candidato.Site,
				Facebook:            candidature.Candidato.Facebook,
				Twitter:             candidature.Candidato.Twitter,
				Instagram:           candidature.Candidato.Instagram,
				Description:         candidature.Descricao,
				Biography:           candidature.Candidato.Biografia,
				PhotoURL:            candidature.Candidato.PhotoURL,
				Party:               candidature.LegendaPartido,
				Name:                candidature.Candidato.Nome,
				BallotName:          candidature.NomeUrna,
				BallotNumber:        int(candidature.NumeroUrna),
				Email:               candidature.Candidato.Email,
			}
			if dbItems[candidature.Municipio] == nil {
				dbItems[candidature.Municipio] = &votingCity{
					City:       candidature.Municipio,
					State:      candidature.UF,
					Candidates: []*candidateForDB{&candidateDataToPersist},
				}
			} else {
				dbItems[candidature.Municipio].Candidates = append(dbItems[candidature.Municipio].Candidates, &candidateDataToPersist)
			}
		}
	}
	return dbItems, nil
}
