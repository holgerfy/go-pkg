package mongo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/holgerfy/go-pkg/app"
	"github.com/holgerfy/go-pkg/config"
	"github.com/holgerfy/go-pkg/funcs"
	"github.com/holgerfy/go-pkg/log"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mode = readpref.Mode

// Mode constants
const (
	PrimaryMode            = readpref.PrimaryMode
	PrimaryPreferredMode   = readpref.PrimaryPreferredMode
	SecondaryMode          = readpref.SecondaryMode
	SecondaryPreferredMode = readpref.SecondaryPreferredMode
	NearestMode            = readpref.NearestMode
)

var (
	client *mongo.Client
	conf   struct {
		URL             string `toml:"url"`
		DbName          string `toml:"database"`
		MaxConnIdleTime int    `toml:"max_conn_idle_time"`
		MaxPoolSize     int    `toml:"max_pool_size"`
		Username        string `toml:"username"`
		Password        string `toml:"password"`
		ReplicaSet      string `toml:"replicaSet"`
		IsSsl           bool   `toml:"is_ssl"`
		CaCert          string `toml:"ca_cert"`
	}
)

type (
	CollectionInfo struct {
		Database   *mongo.Database
		Collection *mongo.Collection
		filter     bson.M
		limit      int64
		skip       int64
		sort       bson.M
		fields     bson.M
	}
)

// Start mongo
func Start() {
	ctx := log.WithFields(context.Background(), map[string]string{"action": "startMongo"})
	log.Logger().Info(ctx, "test ")
	var err error
	err = config.GetInstance().Bind("db", "mongo", &conf)
	if err == config.ErrNodeNotExists {
		return
	}
	mongoOptions := options.Client()
	mongoOptions.SetMaxConnIdleTime(time.Duration(conf.MaxConnIdleTime) * time.Second)
	mongoOptions.SetMaxPoolSize(uint64(conf.MaxPoolSize))
	mongoOptions.SetRetryReads(true)
	mongoOptions.SetRetryWrites(false)
	if conf.Username != "" && conf.Password != "" {
		mongoOptions.SetAuth(options.Credential{Username: conf.Username, Password: conf.Password})
	}
	if conf.IsSsl {
		certs := x509.NewCertPool()
		fmt.Println(funcs.GetRoot() + conf.CaCert)
		if pemData, err := ioutil.ReadFile(funcs.GetRoot() + conf.CaCert); err != nil {
			log.Logger().Info(ctx, "failed to read cert, err: ", err)
			return
		} else {
			certs.AppendCertsFromPEM(pemData)
		}
		tlsConf := &tls.Config{
			RootCAs: certs,
		}
		if funcs.GetEnv() == app.EnvModelLocal {
			tlsConf.InsecureSkipVerify = true
		}
		mongoOptions.SetTLSConfig(tlsConf)
	}

	if conf.ReplicaSet != "" {
		mongoOptions.SetReplicaSet(conf.ReplicaSet)
	}

	client, err = mongo.NewClient(mongoOptions.ApplyURI(conf.URL))
	if err != nil {
		log.Logger().Info(ctx, "self build new client, err: ", err)
		return
	}

	mgoCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Connect(mgoCtx)
	if err != nil {
		log.Logger().Error(ctx, "failed to connect, err: ", err)
	}
}

func Database(name ...string) *CollectionInfo {
	dbName := conf.DbName
	if len(name) == 1 {
		dbName = name[0]
	}

	collection := &CollectionInfo{
		Database: client.Database(dbName),
		filter:   make(bson.M),
	}
	return collection
}

func GetMongoFieldsBsonByString(fields string) bson.M {
	fieldsSlice := strings.Split(fields, ",")
	var res = make(bson.M)
	for _, f := range fieldsSlice {
		f = strings.Replace(f, " ", "", -1)
		res[f] = 1
	}
	return res
}

func DataIsSaveSuccessfully(err error) bool {
	if err == nil {
		return true
	}
	return mongo.IsDuplicateKeyError(err)
}

func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return true
	}
	return mongo.IsDuplicateKeyError(err)
}

func IsNoDocumentErr(err error) bool {
	return err == mongo.ErrNoDocuments
}

func (collection *CollectionInfo) SetTable(name string, mode ...Mode) *CollectionInfo {
	if conf.ReplicaSet != "" {
		rpMode := PrimaryMode
		if len(mode) > 0 && mode[0] > 0 {
			rpMode = mode[0]
		}

		rp, _ := readpref.New(rpMode)
		collection.Collection = collection.Database.Collection(name, options.Collection().SetReadPreference(rp))
	} else {
		collection.Collection = collection.Database.Collection(name)
	}
	return collection
}

// Where  bson.M{"field": "value"}
func (collection *CollectionInfo) Where(m bson.M) *CollectionInfo {
	collection.filter = m
	return collection
}

// Limit
func (collection *CollectionInfo) Limit(n int64) *CollectionInfo {
	collection.limit = n
	return collection
}

// Skip
func (collection *CollectionInfo) Skip(n int64) *CollectionInfo {
	collection.skip = n
	return collection
}

// Sort  bson.M{"create_time":-1}
func (collection *CollectionInfo) Sort(sorts bson.M) *CollectionInfo {
	collection.sort = sorts
	return collection
}

// Fields
func (collection *CollectionInfo) Fields(fields interface{}) *CollectionInfo {
	kind := reflect.TypeOf(fields).Kind()
	if kind == reflect.String {
		fieldStr := fields.(string)
		if fieldStr != "" {
			collection.fields = GetMongoFieldsBsonByString(fieldStr)
		}
	} else if kind == reflect.Map && fields != nil {
		collection.fields = fields.(bson.M)
	}

	return collection
}

// InsertOne
func (collection *CollectionInfo) InsertOne(document interface{}) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	res, err := collection.Collection.InsertOne(ctx, BeforeCreate(document))
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), err
}

func (collection *CollectionInfo) InsertOneOrigin(document interface{}) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	res, err := collection.Collection.InsertOne(ctx, document)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), err
}

// InsertMany
func (collection *CollectionInfo) InsertMany(documents interface{}) ([]string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	var data []interface{}
	data = BeforeCreate(documents).([]interface{})
	res, err := collection.Collection.InsertMany(ctx, data)
	if err != nil {
		return nil, err
	}
	insertedIds := make([]string, 0)
	for _, v := range res.InsertedIDs {
		insertedIds = append(insertedIds, v.(string))
	}
	return insertedIds, nil
}

// UpdateOrInsert documents must contain the _id field
func (collection *CollectionInfo) UpdateOrInsert(documents interface{}) (*mongo.UpdateResult, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	upsert := true
	return collection.Collection.UpdateMany(ctx, bson.M{}, documents, &options.UpdateOptions{Upsert: &upsert})
}

func (collection *CollectionInfo) Upsert(document interface{}) *mongo.SingleResult {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	upsert := true
	var isReturn options.ReturnDocument = 1
	opt := options.FindOneAndUpdateOptions{Upsert: &upsert, ReturnDocument: &isReturn}
	result := collection.Collection.FindOneAndUpdate(ctx, collection.filter, bson.M{"$set": BeforeUpdate(document)}, &opt)
	return result
}

func (collection *CollectionInfo) UpsertByBson(document interface{}) *mongo.SingleResult {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	upsert := true
	var isReturn options.ReturnDocument = 1
	opt := options.FindOneAndUpdateOptions{Upsert: &upsert, ReturnDocument: &isReturn}
	return collection.Collection.FindOneAndUpdate(ctx, collection.filter, document, &opt)
}

// UpdateOne
func (collection *CollectionInfo) UpdateOne(document interface{}) (*mongo.UpdateResult, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return collection.Collection.UpdateOne(ctx, collection.filter, bson.M{"$set": BeforeUpdate(document)})
}

func (collection CollectionInfo) UpByID(id interface{}, document interface{}) (*mongo.UpdateResult, error) {
	return collection.Where(bson.M{"_id": id}).UpdateOne(document)
}

// UpdateMany
func (collection *CollectionInfo) UpdateMany(document interface{}) (*mongo.UpdateResult, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return collection.Collection.UpdateMany(ctx, collection.filter, bson.M{"$set": BeforeUpdate(document)})
}

func (collection *CollectionInfo) UpsertMany(document interface{}) (*mongo.UpdateResult, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	upsert := true
	opt := options.UpdateOptions{Upsert: &upsert}
	return collection.Collection.UpdateMany(ctx, collection.filter, bson.M{"$set": BeforeUpdate(document)}, &opt)
}

// FindOne
func (collection *CollectionInfo) FindOne(document interface{}) error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result := collection.Collection.FindOne(ctx, collection.filter, &options.FindOneOptions{
		Skip:       &collection.skip,
		Sort:       collection.sort,
		Projection: collection.fields,
	})
	return result.Decode(document)
}

func (collection *CollectionInfo) FindByID(id interface{}, document interface{}) error {
	return collection.Where(bson.M{"_id": id}).FindOne(document)
}

// FindMany
func (collection *CollectionInfo) FindMany(documents interface{}) error {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	opt := options.Find().SetProjection(collection.fields).SetLimit(collection.limit).SetSort(collection.sort)
	result, err := collection.Collection.Find(ctx, collection.filter, opt)
	if err != nil {
		return err
	}
	defer result.Close(ctx)
	val := reflect.ValueOf(documents)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return errors.New("result argument must be a slice address")
	}
	slice := reflect.MakeSlice(val.Elem().Type(), 0, 0)

	itemTyp := val.Elem().Type().Elem()
	for result.Next(ctx) {
		item := reflect.New(itemTyp)
		err := result.Decode(item.Interface())
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, reflect.Indirect(item))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if resErr := result.Err(); resErr != nil {
		return resErr
	}

	val.Elem().Set(slice)
	return nil
}

func (collection *CollectionInfo) Delete() (int64, error) {
	if collection.filter == nil || len(collection.filter) == 0 {
		return 0, errors.New("you can't delete all documents, it's very dangerous")
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := collection.Collection.DeleteMany(ctx, collection.filter)
	return result.DeletedCount, err
}

func (collection *CollectionInfo) Count() (int64, error) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return collection.Collection.CountDocuments(ctx, collection.filter)
}

// BeforeCreate
func BeforeCreate(document interface{}) interface{} {
	millis := funcs.GetMillis()
	val := reflect.ValueOf(document)
	typ := reflect.TypeOf(document)
	switch typ.Kind() {
	case reflect.Ptr:
		return BeforeCreate(val.Elem().Interface())

	case reflect.Array, reflect.Slice:
		var sliceData = make([]interface{}, val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			sliceData[i] = BeforeCreate(val.Index(i).Interface()).(bson.M)
		}
		return sliceData

	case reflect.Struct:
		var data = make(bson.M)
		for i := 0; i < typ.NumField(); i++ {
			data[typ.Field(i).Tag.Get("bson")] = val.Field(i).Interface()
		}
		if val.FieldByName("ID").Type() == reflect.TypeOf(primitive.ObjectID{}) {
			data["_id"] = primitive.NewObjectID()
		}

		if val.FieldByName("ID").Kind() == reflect.String && val.FieldByName("ID").Interface() == "" {
			data["_id"] = primitive.NewObjectID().Hex()
		}

		if data["create_time"] == 0 {
			data["create_time"] = millis
		}
		if data["update_time"] == 0 {
			data["update_time"] = millis
		}
		return data

	default:
		if val.Type() == reflect.TypeOf(bson.M{}) {
			if !val.MapIndex(reflect.ValueOf("_id")).IsValid() {
				val.SetMapIndex(reflect.ValueOf("_id"), reflect.ValueOf(primitive.NewObjectID()))
			}
			val.SetMapIndex(reflect.ValueOf("create_time"), reflect.ValueOf(millis))
			val.SetMapIndex(reflect.ValueOf("update_time"), reflect.ValueOf(millis))
		}
		return val.Interface()
	}
}

// BeforeUpdate
func BeforeUpdate(document interface{}) interface{} {
	millis := funcs.GetMillis()
	val := reflect.ValueOf(document)
	typ := reflect.TypeOf(document)
	switch typ.Kind() {
	case reflect.Ptr:
		return BeforeUpdate(val.Elem().Interface())

	case reflect.Array, reflect.Slice:
		var sliceData = make([]interface{}, val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			sliceData[i] = BeforeUpdate(val.Index(i).Interface()).(bson.M)
		}
		return sliceData

	case reflect.Struct:
		var data = make(bson.M)
		for i := 0; i < typ.NumField(); i++ {
			if !isZero(val.Field(i)) {
				tag := strings.Split(typ.Field(i).Tag.Get("bson"), ",")[0]
				data[tag] = val.Field(i).Interface()
				if tag != "_id" {
					data[tag] = val.Field(i).Interface()
				}
			}
		}
		//time.Now().Unix()
		if data["update_time"] == 0 {
			data["update_time"] = time.Now().UnixNano() / 1e6
		}

		return data

	default:
		if val.Type() == reflect.TypeOf(bson.M{}) {
			val.SetMapIndex(reflect.ValueOf("update_time"), reflect.ValueOf(millis))
		}
		return val.Interface()
	}
}

// IsIntn
func IsIntn(p reflect.Kind) bool {
	return p == reflect.Int || p == reflect.Int64 || p == reflect.Uint64 || p == reflect.Uint32
}

func isZero(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func (collection *CollectionInfo) LBS(pipeline interface{}, documents interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Aggregate()
	result, err := collection.Collection.Aggregate(ctx, pipeline, opts)
	fmt.Println(pipeline, err)
	defer result.Close(ctx)
	val := reflect.ValueOf(documents)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return errors.New("result argument must be a slice address")
	}
	slice := reflect.MakeSlice(val.Elem().Type(), 0, 0)
	////fmt.Println(slice)
	itemTyp := val.Elem().Type().Elem()
	for result.Next(ctx) {
		item := reflect.New(itemTyp)
		err := result.Decode(item.Interface())
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, reflect.Indirect(item))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if resErr := result.Err(); resErr != nil {
		return resErr
	}

	val.Elem().Set(slice)
	return nil
}

func (collection *CollectionInfo) Aggregate(pipeline interface{}, documents interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Aggregate()
	result, _ := collection.Collection.Aggregate(ctx, pipeline, opts)
	defer result.Close(ctx)
	val := reflect.ValueOf(documents)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return errors.New("result argument must be a slice address")
	}
	slice := reflect.MakeSlice(val.Elem().Type(), 0, 0)
	itemTyp := val.Elem().Type().Elem()
	for result.Next(ctx) {
		item := reflect.New(itemTyp)
		err := result.Decode(item.Interface())
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, reflect.Indirect(item))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if resErr := result.Err(); resErr != nil {
		return resErr
	}

	val.Elem().Set(slice)
	return nil
}
