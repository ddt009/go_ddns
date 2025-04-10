#!/bin/sh
# 报告IPv6地址
REPORT_URL="https://example.com/"
HOST_NAME="YOUR-HOST-NAME"
INTERFACE="wan"    # 网卡名称 eth0 / wan
ipv6=$(ip -6 addr show ${INTERFACE} | grep -v deprecated | grep 'inet6 [^f:]' | awk -F' ' '{print $2}' | grep /64 | awk -F'/' '{print $1}' | tail -1)
echo "IPv6地址: $ipv6"
if [ -z "$ipv6" ]; then 
    exit 0
fi
if [ ! -f .ipv6 ]; then
    touch .ipv6
fi
old_ipv6=$(cat .ipv6)
if [ "$ipv6" != "$old_ipv6" ]; then
    echo "$ipv6" > .ipv6
    curl -s -X POST "${REPORT_URL}" -H "Content-Type: application/json" -d "{\"host\": \"${HOST_NAME}\",\"ipv6\":\"${ipv6}\"}"
    # echo "curl -s -X POST \"${REPORT_URL}\" -H \"Content-Type: application/json\" -d \"{\\\"host\\\": \\\"${HOST_NAME}\\\",\\\"ipv6\\\":\\\"${ipv6}\\\"}\""
fi
