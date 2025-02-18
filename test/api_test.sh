#!/bin/bash

API_URL="http://localhost:8080/api/v1"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "开始API测试..."

# 注册节点
echo -e "\n${GREEN}测试注册节点${NC}"
NODE_RESPONSE=$(curl -s -X POST "${API_URL}/nodes" \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "test-node-1",
    "ip": "192.168.1.100",
    "resources": {
      "total_cpu": 8,
      "used_cpu": 0,
      "total_memory": 16384,
      "used_memory": 0,
      "total_storage": 1024000,
      "used_storage": 0
    }
  }')
echo "节点注册响应: $NODE_RESPONSE"
NODE_ID=$(echo $NODE_RESPONSE | jq -r '.id')

# 获取节点列表
echo -e "\n${GREEN}测试获取节点列表${NC}"
curl -s "${API_URL}/nodes" | jq '.'

# 创建Docker实例
echo -e "\n${GREEN}测试创建Docker实例${NC}"
CREATE_RESPONSE=$(curl -s -X POST "${API_URL}/instances" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-nginx",
    "type": "docker",
    "metadata": {
      "image": "nginx:latest"
    }
  }')
echo "创建响应: $CREATE_RESPONSE"
INSTANCE_ID=$(echo $CREATE_RESPONSE | jq -r '.ID')

# 获取实例列表
echo -e "\n${GREEN}测试获取实例列表${NC}"
curl -s "${API_URL}/instances" | jq '.'

# 获取特定实例
echo -e "\n${GREEN}测试获取特定实例${NC}"
curl -s "${API_URL}/instances/${INSTANCE_ID}" | jq '.'

# 删除实例
echo -e "\n${GREEN}测试删除实例${NC}"
DELETE_STATUS=$(curl -s -w "%{http_code}" -X DELETE "${API_URL}/instances/${INSTANCE_ID}")
if [ "$DELETE_STATUS" -eq 204 ]; then
    echo "实例删除成功"
else
    echo -e "${RED}实例删除失败: $DELETE_STATUS${NC}"
fi

echo -e "\n${GREEN}API测试完成${NC}"