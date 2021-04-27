package consts

import "time"

//TODO: (low priority)move some of these consts to config file
const PageSize = 30
const MsgPageSize = 10
const SystemLoadThreshold float64 = 5.0
const TokenExpireDays = 31
const WanderPageSize = 15
const MaxPage = 150
const SearchPageSize = 30
const SearchMaxPage = 100
const SearchMaxLength = 30
const PostMaxLength = 10000
const ImageMaxWidth = 10000
const ImageMaxHeight = 10000
const VoteOptionMaxCharacters = 15
const VoteMaxOptions = 4
const MaxDevicesPerUser = 6
const ReportMaxLength = 1000
const ImgMaxLength = 2000000
const Base64Rate = 1.33333333
const AesIv = "12345678901234567890123456789012"

const PushApiLogFile = "push.log"
const ServicesApiLogFile = "services-api.log"
const DetailLogFile = "detail-log.log"
const SecurityApiLogFile = "security-api.log"

const DatabaseReadFailedString = "数据库读取失败，请联系管理员"
const DatabaseWriteFailedString = "数据库写入失败，请联系管理员"
const DatabaseDamagedString = "数据库损坏，请联系管理员"
const DatabaseEncryptFailedString = "数据库加密失败，请联系管理员"

const DzName = "洞主"
const ExtraNamePrefix = "You Win "

var TimeLoc, _ = time.LoadLocation("Asia/Shanghai")

var Names0 = []string{
	"Angry",
	"Baby",
	"Crazy",
	"Diligent",
	"Excited",
	"Fat",
	"Greedy",
	"Hungry",
	"Interesting",
	"Jolly",
	"Kind",
	"Little",
	"Magic",
	"Naïve",
	"Old",
	"PKU",
	"Quiet",
	"Rich",
	"Superman",
	"Tough",
	"Undefined",
	"Valuable",
	"Wifeless",
	"Xiangbuchulai",
	"Young",
	"Zombie",
}

var Names1 = []string{
	"Alice",
	"Bob",
	"Carol",
	"Dave",
	"Eve",
	"Francis",
	"Grace",
	"Hans",
	"Isabella",
	"Jason",
	"Kate",
	"Louis",
	"Margaret",
	"Nathan",
	"Olivia",
	"Paul",
	"Queen",
	"Richard",
	"Susan",
	"Thomas",
	"Uma",
	"Vivian",
	"Winnie",
	"Xander",
	"Yasmine",
	"Zach",
}
