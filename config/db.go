package config

import (
	"os"

	"github.com/gocroot/helper/atdb"
)

var MongoString string = os.Getenv("MONGOSTRING")

var MongoStringGeo string = os.Getenv("MONGOSTRINGGEO")

var MongoStringGeoVillage string = os.Getenv("MONGOSTRINGGEOVILLAGE")

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
