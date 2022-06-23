package funcs

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//if version1 > version2 return 1; if version1 < version2 return -1; other: 0。
func CompareVersion(version1 string, version2 string) int {
	var res int
	ver1Strs := strings.Split(version1, ".")
	ver2Strs := strings.Split(version2, ".")
	ver1Len := len(ver1Strs)
	ver2Len := len(ver2Strs)
	verLen := ver1Len
	if len(ver1Strs) < len(ver2Strs) {
		verLen = ver2Len
	}
	for i := 0; i < verLen; i++ {
		var ver1Int, ver2Int int
		if i < ver1Len {
			ver1Int, _ = strconv.Atoi(ver1Strs[i])
		}
		if i < ver2Len {
			ver2Int, _ = strconv.Atoi(ver2Strs[i])
		}
		if ver1Int < ver2Int {
			res = -1
			break
		}
		if ver1Int > ver2Int {
			res = 1
			break
		}
	}
	return res
}

func SubSlice(ori, src []string) []string {
	res := make([]string, 0)
	temp := make(map[string]struct{})
	for _, v := range src {
		if _, ok := temp[v]; !ok {
			temp[v] = struct{}{}
		}
	}
	for _, v := range ori {
		if _, ok := temp[v]; !ok {
			res = append(res, v)
		}
	}
	return res
}

func HttpBuildQuery(params map[string]interface{}) (paramStr string) {
	paramsArr := make([]string, 0, len(params))
	for k, v := range params {
		paramsArr = append(paramsArr, fmt.Sprintf("%s=%s", k, v))
	}
	paramStr = strings.Join(paramsArr, "&")
	return paramStr
}

func RemoteIp(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get("XRealIP"); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get("XForwardedFor"); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

func HasLocalIPAddr(ip string) bool {
	return HasLocalIP(net.ParseIP(ip))
}

func HasLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	return ip4[0] == 10 || // 10.0.0.0/8
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || // 172.16.0.0/12
		(ip4[0] == 169 && ip4[1] == 254) || // 169.254.0.0/16
		(ip4[0] == 192 && ip4[1] == 168) // 192.168.0.0/16
}

func GetNanos() int64 {
	return time.Now().UnixNano()
}

func GetMillis() int64 {
	return GetNanos() / 1e6
}

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func Md516(str string) string {
	res := Md5(str)
	return res[8:24]
}

func SHA1(s string) string {
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}

func StrSha256(str string) string {
	hashInBytes := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hashInBytes[:])
}

func FilterMapByKeys(data map[string]interface{}, keys []string) map[string]interface{} {
	var res map[string]interface{}
	for _, key := range keys {
		if _, ok := data[key]; ok {
			res[key] = data[key]
		}
	}
	return res
}

func FilterArrayByKeys(data []map[string]interface{}, keys []string) []map[string]interface{} {
	var res []map[string]interface{}
	for _, m := range data {
		var cm map[string]interface{}
		for _, key := range keys {
			if _, ok := m[key]; ok {
				cm[key] = m[key]
			}
		}
		res = append(res, cm)
	}
	return res
}

func GetRandString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result[i] = bytes[r.Intn(len(bytes))]
	}
	return string(result)
}

func JsonEncode(data interface{}) string {
	json, _ := json.Marshal(data)
	return string(json)
}

func JsonDecode(data string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(data), &result)
	return result
}

func StructToMap(obj interface{}) map[string]interface{} {
	obj1 := reflect.TypeOf(obj)
	obj2 := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < obj1.NumField(); i++ {
		data[obj1.Field(i).Name] = obj2.Field(i).Interface()
	}
	return data
}

func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

func InArray(val string, arr []interface{}) bool {
	for _, v := range arr {
		if val == v {
			return true
		}
	}
	return false
}

func GetDate() int32 {
	date, _ := strconv.ParseInt(time.Now().Format("20060102"), 10, 32)
	return int32(date)
}

func DesensitizeStr(str string) string {
	len := len(str)
	if len <= 4 {
		return str
	} else if len > 4 && len < 9 {
		return str[0:2] + "**" + str[len-2:]
	} else {
		return str[0:3] + "****" + str[len-4:]
	}
}

func PanicTrace(err interface{}) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v\n", err)
	for i := 0; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fmt.Fprintf(buf, "%s:%d (0x%x) \n", file, line, pc)
	}
	return buf.String()
}

func ScanDir(dirName string) ([]string, error) {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		return nil, err
	}
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, dirName+string(os.PathSeparator)+file.Name())
	}
	return fileList, nil
}

func MergeMap(x, y map[string]interface{}) map[string]interface{} {
	n := make(map[string]interface{})
	for i, v := range x {
		for j, w := range y {
			if i == j {
				n[i] = w
			} else {
				if _, ok := n[i]; !ok {
					n[i] = v
				}
				if _, ok := n[j]; !ok {
					n[j] = w
				}
			}
		}
	}
	return n
}

func RemoveDuplicatesAndEmpty(arr []string) (ret []string) {
	sort.Strings(arr)
	for i := 0; i < len(arr); i++ {
		if (i > 0 && arr[i-1] == arr[i]) || len(arr[i]) == 0 {
			continue
		}
		ret = append(ret, arr[i])
	}
	return
}

func GetRoot() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return strings.Replace(dir, "\\", "/", -1)
}

func GetEnv() string {
	return os.Getenv("RUN_ENV")
}

func HmacSha256(src, key string) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(src))
	return hex.EncodeToString(m.Sum(nil))
}

func Millis2FitTimeSpan(millis int) string {
	if 1000 <= millis && millis < 3600000 {
		var min = millis / 1000 % 3600 / 60
		var sec = millis / 1000 % 60
		return fmt.Sprintf("%02d:%02d", min, sec)
	}
	var hor = millis / 1000 / 3600
	var min = millis / 1000 % 3600 / 60
	var sec = millis / 1000 % 60
	return fmt.Sprintf("%02d:%02d:%02d", hor, min, sec)
}
