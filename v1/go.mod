module tracker_v1

go 1.13

require (
	github.com/go-sql-driver/mysql v1.5.0
	github.com/hashicorp/memberlist v0.2.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)

replace github.com/hashicorp/memberlist => github.com/0xjiayu/memberlist v0.2.0
