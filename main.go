//go:generate protoc --go_out=. --go_opt=paths=source_relative internal/proto/appsinstalled.proto
package main

import (
	"flag"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"go_memcache_loader/internal/loader"
	appsproto "go_memcache_loader/internal/proto"
)

func protoTest() {
	sample := "idfa\t1rfw452y52g2gq4g\t55.55\t42.42\t1423,43,567,3,7,23\ngaid\t7rfw452y52g2gq4g\t55.55\t42.42\t7423,424"

	for _, line := range strings.Split(sample, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) < 5 {
			continue
		}

		devType, devID, latStr, lonStr, rawApps := parts[0], parts[1], parts[2], parts[3], parts[4]

		// Parse applications
		var apps []uint32
		for _, app := range strings.Split(rawApps, ",") {
			if app == "" {
				continue
			}
			id, err := strconv.ParseUint(app, 10, 32)
			if err != nil {
				log.Printf("Invalid app ID: %s", app)
				continue
			}
			apps = append(apps, uint32(id))
		}

		// Parse coordinates
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			log.Fatalf("Invalid lat: %v", err)
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			log.Fatalf("Invalid lon: %v", err)
		}

		// Create a protobuf message
		ua := &appsproto.UserApps{
			Lat:  &lat,
			Lon:  &lon,
			Apps: apps,
		}

		// Serialization
		packed, err := proto.Marshal(ua)
		if err != nil {
			log.Fatalf("Marshaling error: %v", err)
		}

		// Deserialization
		unpacked := &appsproto.UserApps{}
		if err := proto.Unmarshal(packed, unpacked); err != nil {
			log.Fatalf("Unmarshaling error: %v", err)
		}

		// Comparing the results
		if !proto.Equal(ua, unpacked) {
			log.Fatalf("Protobuf mismatch:\nOriginal: %v\nUnpacked: %v", ua, unpacked)
		}

		log.Printf("Successfully processed data for %s:%s\n", devType, devID)
	}
}

func main() {
	var (
		test      = flag.Bool("test", false, "Run proto test")
		dryRun    = flag.Bool("dry", false, "Dry run (no memcached writes)")
		pattern   = flag.String("pattern", "./data/appsinstalled/*.tsv.gz", "File pattern")
		idfa      = flag.String("idfa", "127.0.0.1:33013", "IDFA memcached address")
		gaid      = flag.String("gaid", "127.0.0.1:33014", "GAID memcached address")
		adid      = flag.String("adid", "127.0.0.1:33015", "ADID memcached address")
		dvid      = flag.String("dvid", "127.0.0.1:33016", "DVID memcached address")
		workers   = flag.Int("workers", loader.GetDefaultWorkers(), "Number of workers")
		batchSize = flag.Int("batch-size", loader.GetDefaultBatchSize(), "Batch size")
	)
	flag.Parse()

	if *test {
		protoTest()
		return
	}

	deviceMemc := map[string]string{
		"idfa": *idfa,
		"gaid": *gaid,
		"adid": *adid,
		"dvid": *dvid,
	}

	ld := loader.NewLoader(*dryRun)

	files, err := filepath.Glob(*pattern)
	if err != nil {
		log.Fatalf("Failed to glob files: %v", err)
	}

	for _, fn := range files {
		processed, errors := ld.ProcessFile(fn, deviceMemc, *workers, *batchSize)
		if processed == 0 {
			if err := loader.DotRename(fn); err != nil {
				log.Printf("Failed to rename %s: %v", fn, err)
			}
			continue
		}

		errRate := float64(errors) / float64(processed)
		if errRate < loader.GetNormalErrRate() {
			log.Printf("Acceptable error rate (%.4f). Successfully loaded %d records", errRate, processed)
		} else {
			log.Printf("High error rate (%.4f > %.4f). Failed load", errRate, loader.GetNormalErrRate())
		}

		if err := loader.DotRename(fn); err != nil {
			log.Printf("Failed to rename %s: %v", fn, err)
		}
	}
}
