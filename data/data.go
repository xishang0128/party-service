package data

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/metacubex/bbolt"
)

var (
	filePath    string
	initOnce    sync.Once
	bucketData  = []byte("data")
	defaultData *data
)

type data struct {
	DB *bbolt.DB
}

func initData() {
	db, err := bbolt.Open(filePath, 0o666, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		if os.IsNotExist(err) || err == bbolt.ErrInvalid || err == bbolt.ErrChecksum || err == bbolt.ErrVersionMismatch {
			if os.Remove(filePath) == nil {
				log.Printf("Removed invalid data file: %s", filePath)
			}
			db, err = bbolt.Open(filePath, 0o666, &bbolt.Options{Timeout: time.Second})
		}
		if err != nil {
			log.Fatalf("Error opening data file: %s", err)
		}
	}
	defaultData = &data{DB: db}
}

func (d *data) Set(k, v string) {
	if d.DB == nil {
		return
	}
	err := d.DB.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketData)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(k), []byte(v))
	})
	if err != nil {
		log.Printf("Failed to write data for %s: %s", k, err)
	}
}

func (d *data) Get(k string) (v string) {
	if d.DB == nil {
		return ""
	}
	d.DB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bucketData)
		if bucket != nil {
			v = string(bucket.Get([]byte(k)))
		}
		return nil
	})
	return v
}

func Data() *data {
	initOnce.Do(initData)

	return defaultData
}

func SetFilePath(path string) {
	filePath = path
}
