package config

import (
	"os"

	"github.com/gocroot/helper/atdb"
)

var MongoString string = os.Getenv("MONGOSTRING")

var MongoStringGeo string = "mongodb+srv://ayalarifki:Edqt6j5LplXRs5OH@appabsensi.lnfmk5s.mongodb.net/"

var MongoStringGeoVillage string = "mongodb+srv://farhan350411:Ge3S8IS6qP6gT3CC@cluster0.vyo74.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"

var mongoinfo = atdb.DBInfo{
	DBString: MongoString,
	DBName:   "jualin",
}

var Mongoconn, ErrorMongoconn = atdb.MongoConnect(mongoinfo)

var MongoInfoGeo = atdb.DBInfo{
	DBString: MongoStringGeo,
	DBName:   "ayala-crea",
}

var MongoconnGeo, ErrorMongoconnGeo = atdb.MongoConnect(MongoInfoGeo)

var MongoInfoGeoVillage = atdb.DBInfo{
	DBString: MongoStringGeoVillage,
	DBName:   "gis",
}

var MongoconnGeoVill, ErrorMongoconnGeoVill = atdb.MongoConnect(MongoInfoGeoVillage)
