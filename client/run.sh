#对内网端口：20080 -> 公共端口对外：80
nohup ./ferry_client -server quickex.com.cn:20080 -client :80 >/dev/null 2>&1 &
