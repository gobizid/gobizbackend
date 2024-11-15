package config

import (
	"os"

	"github.com/gocroot/helper/atdb"
)

var MongoString string = os.Getenv("MONGOSTRING")

var MongoStringGeo string = "mongodb+srv://ayalarifki:Edqt6j5LplXRs5OH@appabsensi.lnfmk5s.mongodb.net/"

var mongoinfo = atdb.DBInfo{
	DBString: MongoString,
	DBName:   "jualin",
}

var Mongoconn, ErrorMongoconn = atdb.MongoConnect(mongoinfo)

var MongoInfoGeo = atdb.DBInfo{
	DBString: MongoStringGeo,
	DBName:   "geo",
}

var MongoconnGeo, ErrorMongoconnGeo = atdb.MongoConnect(MongoInfoGeo)
