#对内网端口：20080 -> 公共端口对外：10080
nohup ./ferry_server -server :20080 -public :10080 >/dev/null 2>&1 &