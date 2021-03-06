package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/candidatos-info/descritor"
	"github.com/gocarina/gocsv"
	"github.com/golang/protobuf/proto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/text/encoding/charmap"
	"google.golang.org/api/drive/v3"
)

const (
	pageSize = 1000
)

var (
	re       = regexp.MustCompile(`([0-9]+)`) // regexp to extract numbers
	rolesMap = map[string]string{
		"LM":  "vereador",
		"EM":  "prefeito",
		"VEM": "vice-prefeito",
	}
)

type gDriveCandFiles struct {
	candidatureFile *drive.File
	picture         *drive.File
}

type pathResolver struct {
	GoogleDriveID      string `csv:"google_drive_id"` // ID do arquivo no Google Drive
	ProtoBuffLocalPath string `csv:"proto_buff_path"` // Path para o arquivo proto buff armazenado localmente
}

type pictureReference struct {
	GoogleDriveID   string `csv:"google_drive_id"`   // ID do arquivo de foto no Google Drive
	TSESequencialID string `csv:"tse_sequencial_id"` // ID sequencial do candidato no TSE
}

func main() {
	pathsFile := flag.String("candidaturesPaths", "", "arquivo contendo os paths dos arquivos de candidaturas locais e no Google Drive")
	picturesFile := flag.String("picturesReferences", "", "arquivo contento referência dos arquivos de fotos processados")
	state := flag.String("state", "", "estado para ser enriquecido")
	dbURL := flag.String("dbURL", "", "URL de conexão com banco MongoDB")
	dbName := flag.String("dbName", "", "nome do banco de dados")
	googleDriveCredentialsFile := flag.String("credentials", "", "chave de credenciais o Goodle Drive")
	goodleDriveOAuthTokenFile := flag.String("OAuthToken", "", "arquivo com token oauth")
	offset := flag.Int("offset", 0, "offset que aponta para a linha de início do processamento")
	flag.Parse()
	if *pathsFile == "" {
		log.Fatal("informe o path para o arquivo contendo os paths dos protocol buffers")
	}
	if *dbURL == "" {
		log.Fatal("informe a URL de conexão com banco MongoDB")
	}
	if *dbName == "" {
		log.Fatal("informe o nome do banco de dados")
	}
	if *state == "" {
		log.Fatal("informe o estado a ser processado")
	}
	if *googleDriveCredentialsFile == "" {
		log.Fatal("informe o path para o arquivo de credenciais do Goodle Drive")
	}
	if *goodleDriveOAuthTokenFile == "" {
		log.Fatal("informe o path para o arquivo de token OAuth do Google Drive")
	}
	if *offset < 0 {
		log.Fatal("offset deve ser maior ou igual a zero")
	}
	if *picturesFile == "" {
		log.Fatal("informe o path para o arquivo contendo as referências das fotos de candidaturas")
	}
	// creating connection with MongoDB
	c, err := New(*dbURL, *dbName)
	if err != nil {
		log.Fatalf("failed to connect with data base: %v\n", err)
	}
	// Creating Google Drive client
	googleDriveService, err := createGoogleDriveClient(*googleDriveCredentialsFile, *goodleDriveOAuthTokenFile)
	if err != nil {
		log.Fatalf("falha ao criar cliente do Google Drive, erro %q", err)
	}
	if err := summarize(*pathsFile, *state, *picturesFile, c, googleDriveService, *offset); err != nil {
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

func summarize(pathsFile, state, picturesFile string, dbClient *Client, googleDriveService *drive.Service, offset int) error {
	processedPicturesCache, err := getProcessedPicturesCache(picturesFile)
	if err != nil {
		return err
	}
	pathsResolver, err := getPathsResolverFromFile(pathsFile)
	if err != nil {
		return err
	}
	sort.Slice(pathsResolver, func(i, j int) bool { // sorting list using sequencial ID gotten from local path
		prevIndex, err := strconv.Atoi(re.FindAllString(filepath.Base(pathsResolver[i].ProtoBuffLocalPath), -1)[0])
		if err != nil {
			log.Fatalf("falha ao converter o ID do Google Drive [%s] para inteiro, erro %v", pathsResolver[i].GoogleDriveID, err)
		}
		nextIndex, err := strconv.Atoi(re.FindAllString(filepath.Base(pathsResolver[j].ProtoBuffLocalPath), -1)[0])
		if err != nil {
			log.Fatalf("falha ao converter o ID do Google Drive [%s] para inteiro, erro %v", pathsResolver[j].GoogleDriveID, err)
		}
		return prevIndex < nextIndex
	})
	nextOffset := offset
	citiesMap := make(map[string]struct{})
	for _, pathResolver := range pathsResolver[offset:] {
		candidate, err := pathResolverToCandidature(pathResolver, googleDriveService, processedPicturesCache)
		if err != nil {
			return fmt.Errorf("falha ao deserializar dados de candidatura. OFFSET: [%d], erro %v", nextOffset, err)
		}
		if _, err := dbClient.SaveCandidate(candidate); err != nil {
			return fmt.Errorf("falha ao salvar candidatos no banco. OFFSET: [%d], erro %v", nextOffset, err)
		}
		log.Printf("saved candidate [%s] on db\n", candidate.SequencialCandidate)
		citiesMap[candidate.City] = struct{}{}
		nextOffset++
	}
	var cities []string
	for city := range citiesMap {
		cities = append(cities, city)
	}
	location := &descritor.Location{
		State:  state,
		Cities: cities,
	}
	if _, err := dbClient.SaveLocation(location); err != nil {
		return fmt.Errorf("falha ao salvar local de votação no banco. OFFSET: [%d], erro %v", nextOffset, err)
	}
	fmt.Printf("saved location for state [%s] on db\n", state)
	return nil
}

func getProcessedPicturesCache(picturesFile string) (map[string]string, error) {
	file, err := os.Open(picturesFile)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo [%s] contendo os cache de fotos processadas, erro %v", picturesFile, err)
	}
	defer file.Close()
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		// Enforcing reading the TSE zip file as ISO 8859-1 (latin 1)
		r := csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(in))
		r.LazyQuotes = true
		r.Comma = ','
		return r
	})
	var pc []*pictureReference
	if err := gocsv.UnmarshalFile(file, &pc); err != nil {
		return nil, fmt.Errorf("falha ao inflar slice de referência de fotos [%s], erro %v", picturesFile, err)
	}
	cache := make(map[string]string)
	for _, r := range pc {
		cache[r.TSESequencialID] = r.GoogleDriveID
	}
	return cache, nil
}

func pathResolverToCandidature(pathResolver *pathResolver, googleDriveService *drive.Service, processedPicturesCache map[string]string) (*descritor.CandidateForDB, error) {
	var bytes []byte
	var err error
	bytes, err = ioutil.ReadFile(pathResolver.ProtoBuffLocalPath) // trying to read bytes from local file
	if err != nil {                                               // if reading local file failed, try to read from file on Google Drive
		bytes, err = fetchCandidatureBytesFromGoogleDrive(pathResolver, googleDriveService)
		if err != nil {
			return nil, err
		}
	}
	var candidature descritor.Candidatura
	if err = proto.Unmarshal(bytes, &candidature); err != nil {
		return nil, fmt.Errorf("falha ao deserializar bytes de arquivo de candidatura com ID [%s] para struct descritor.Candidatura, erro %v", pathResolver.GoogleDriveID, err)
	}
	if _, ok := processedPicturesCache[candidature.SequencialCandidato]; ok {
		candidature.Candidato.PhotoURL = fmt.Sprintf("https://drive.google.com/uc?id=%s&export=download", processedPicturesCache[candidature.SequencialCandidato])
	} else {
		candidature.Candidato.PhotoURL = "https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_640.png"
	}
	return &descritor.CandidateForDB{
		Gender:              candidature.Candidato.Genero,
		SequencialCandidate: candidature.SequencialCandidato,
		Description:         candidature.Descricao,
		Biography:           candidature.Candidato.Biografia,
		PhotoURL:            candidature.Candidato.PhotoURL,
		Party:               candidature.LegendaPartido,
		Name:                candidature.Candidato.Nome,
		BallotName:          candidature.NomeUrna,
		BallotNumber:        int(candidature.NumeroUrna),
		Email:               candidature.Candidato.Email,
		Role:                rolesMap[candidature.Cargo],
		Year:                int(candidature.Legislatura),
		City:                candidature.Municipio,
		State:               candidature.UF,
	}, nil
}

func fetchCandidatureBytesFromGoogleDrive(pathResolver *pathResolver, googleDriveService *drive.Service) ([]byte, error) {
	r, err := googleDriveService.Files.Get(pathResolver.GoogleDriveID).Download()
	if err != nil {
		return nil, fmt.Errorf("falha ao trazer bytes de arquivo com ID [%s] do Google Drive, erro %v", pathResolver.GoogleDriveID, err)
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler bytes de arquivo de ID [%s] trazido do Google Drive, erro %q", pathResolver.GoogleDriveID, err)
	}
	time.Sleep(time.Second * 1) // esse delay é colocado para evitar atingir o limite de requests por segundo. Preste atenção ao tamanho do arquivo que irá enviar.
	return b, nil
}

func getPathsResolverFromFile(pathsFile string) ([]*pathResolver, error) {
	file, err := os.Open(pathsFile)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo [%s] contendo os paths dos protocol buffers, erro %v", pathsFile, err)
	}
	defer file.Close()
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		// Enforcing reading the TSE zip file as ISO 8859-1 (latin 1)
		r := csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(in))
		r.LazyQuotes = true
		r.Comma = ','
		return r
	})
	var paths []*pathResolver
	if err := gocsv.UnmarshalFile(file, &paths); err != nil {
		return nil, fmt.Errorf("falha ao inflar slice de paths de arquivos protocol buffers local [%s], erro %v", pathsFile, err)
	}
	return paths, nil
}
