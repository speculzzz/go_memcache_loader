package loader

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	appsproto "go_memcache_loader/internal/proto"

	"google.golang.org/protobuf/proto"

	"github.com/bradfitz/gomemcache/memcache"
)

const (
	defaultWorkers     = 4
	defaultBatchSize   = 100
	defaultMemcTimeOut = 3
	normalErrRate      = 0.01
)

type AppsInstalled struct {
	DevType string
	DevID   string
	Lat     float64
	Lon     float64
	Apps    []uint32
}

type Loader struct {
	clients    map[string]*memcache.Client
	clientsMtx sync.Mutex
	dryRun     bool
}

func NewLoader(dryRun bool) *Loader {
	return &Loader{
		clients: make(map[string]*memcache.Client),
		dryRun:  dryRun,
	}
}

func GetDefaultWorkers() int {
	return defaultWorkers
}

func GetDefaultBatchSize() int {
	return defaultBatchSize
}

func GetDefaultMemcTimeOut() int {
	return defaultMemcTimeOut
}

func GetNormalErrRate() float64 {
	return normalErrRate
}

func (l *Loader) getClient(addr string) *memcache.Client {
	l.clientsMtx.Lock()
	defer l.clientsMtx.Unlock()

	if client, ok := l.clients[addr]; ok {
		return client
	}

	log.Printf("Create connection to %s", addr)
	client := memcache.New(addr)
	client.Timeout = defaultMemcTimeOut * time.Second
	l.clients[addr] = client
	return client
}

func (l *Loader) insertApp(addr string, app *AppsInstalled) bool {
	ua := &appsproto.UserApps{
		Lat:  &app.Lat,
		Lon:  &app.Lon,
		Apps: app.Apps,
	}
	key := fmt.Sprintf("%s:%s", app.DevType, app.DevID)
	packed, err := proto.Marshal(ua)
	if err != nil {
		log.Printf("Failed to marshal UserApps: %v", err)
		return false
	}

	if l.dryRun {
		log.Printf("%s - %s -> %+v", addr, key, ua)
		return true
	}

	client := l.getClient(addr)
	err = client.Set(&memcache.Item{
		Key:   key,
		Value: packed,
	})
	if err != nil {
		log.Printf("Failed to set %s: %v", key, err)
		return false
	}

	return true
}

func parseAppsInstalled(line string) *AppsInstalled {
	parts := strings.Split(line, "\t")
	if len(parts) < 5 {
		return nil
	}

	devType, devID, latStr, lonStr, rawApps := parts[0], parts[1], parts[2], parts[3], parts[4]
	if devType == "" || devID == "" {
		return nil
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		log.Printf("Invalid lat: %s", line)
		return nil
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		log.Printf("Invalid lon: %s", line)
		return nil
	}

	apps := make([]uint32, 0)
	for _, a := range strings.Split(rawApps, ",") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		appID, err := strconv.ParseInt(a, 10, 32)
		if err != nil {
			log.Printf("Invalid appID: %s", a)
			continue
		}
		apps = append(apps, uint32(appID))
	}

	return &AppsInstalled{
		DevType: devType,
		DevID:   devID,
		Lat:     lat,
		Lon:     lon,
		Apps:    apps,
	}
}

func (l *Loader) ProcessFile(filename string, deviceMemc map[string]string, workers int, batchSize int) (processed int, errors int) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open %s: %v", filename, err)
		return 0, 0
	}
	defer file.Close()

	log.Printf("Processing %s", filename)

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("Failed to create gzip reader: %v", err)
		return 0, 0
	}
	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	taskCh := make(chan *AppsInstalled, workers*batchSize)
	resultCh := make(chan bool, workers*batchSize)

	var wg sync.WaitGroup
	wg.Add(workers)

	// Start workers
	for i := 0; i < workers; i++ {
		go func(n int) {
			defer wg.Done()
			log.Printf("w%d: started", n)
			for app := range taskCh {
				addr := deviceMemc[app.DevType]
				if addr == "" {
					log.Printf("Unknown device type: %s", app.DevType)
					resultCh <- false
					continue
				}
				resultCh <- l.insertApp(addr, app)
			}
			log.Printf("w%d: stopped", n)
		}(i)
	}

	// Read a file and submit tasks
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			app := parseAppsInstalled(line)
			if app == nil {
				errors++
				continue
			}

			taskCh <- app
		}
		close(taskCh)
	}()

	// Collecting Results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for ok := range resultCh {
		if ok {
			processed++
		} else {
			errors++
		}
	}

	log.Printf("Done %s: processed=%d errors=%d", filename, processed, errors)

	return processed, errors
}

func DotRename(path string) error {
	dir, file := filepath.Split(path)
	newPath := filepath.Join(dir, "."+file)
	return os.Rename(path, newPath)
}
