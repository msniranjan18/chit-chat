#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== ChitChat API Test ===${NC}\n"

# Base URL
BASE_URL="http://localhost:8080"

# User Configuration
USER1_NAME="Manav"
USER1_PHONE="9575417548"

USER2_NAME="Sami"
USER2_PHONE="8120890722"

# Global variables to track state
CHAT_ID=""
MESSAGE_ID=""

# Check if jq is available
if command -v jq &> /dev/null; then
    JSON_PARSER="jq"
else
    JSON_PARSER="python"
fi

# Function to parse JSON response
parse_json() {
    local json="$1"
    local key="$2"
    
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$json" | jq -r ".$key" 2>/dev/null
    else
        python3 -c "import sys, json; data = json.load(sys.stdin); keys = '$key'.split('.'); result = data; [result := result.get(k) for k in keys if isinstance(result, dict)]; print(result if result is not None else '')" <<< "$json" 2>/dev/null
    fi
}

# Function to register a user
register_user() {
    local name=$1
    local phone=$2
    local upper_name=$(echo "$name" | tr '[:lower:]' '[:upper:]')
    
    echo -e "${GREEN}Registering ${name}...${NC}"
    
    local response
    response=$(curl -s -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$phone\",\"name\":\"$name\"}")
    
    if echo "$response" | grep -q "error\|Error\|failed\|Failed"; then
        echo -e "${RED}Registration failed for ${name}${NC}"
        return 1
    fi
    
    # Extract data
    local token=$(parse_json "$response" "token")
    local user_id=$(parse_json "$response" "user.id")
    
    # If standard parse fails, try regex backup
    [ -z "$token" ] && token=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    [ -z "$user_id" ] && user_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # STORAGE: Using dynamic variable names to avoid "declare -A" issues
    eval "${upper_name}_TOKEN=\"$token\""
    eval "${upper_name}_ID=\"$user_id\""
    
    echo -e "  ${name} Token: ${token:0:30}..."
    echo -e "  ${name} ID: $user_id"
    return 0
}

# Function to search for user
search_user() {
    local searcher_name=$1
    local target_name=$2
    local searcher_upper=$(echo "$searcher_name" | tr '[:lower:]' '[:upper:]')
    eval "local token=\$${searcher_upper}_TOKEN"
    
    echo -e "\n${GREEN}${searcher_name} searching for ${target_name}...${NC}"
    curl -s -X GET "$BASE_URL/api/users/search?q=${target_name}" \
        -H "Authorization: Bearer $token" | (jq . 2>/dev/null || cat)
}

# Function to create chat
create_chat() {
    local creator_name=$1
    local other_name=$2
    local creator_upper=$(echo "$creator_name" | tr '[:lower:]' '[:upper:]')
    local other_upper=$(echo "$other_name" | tr '[:lower:]' '[:upper:]')
    
    eval "local token=\$${creator_upper}_TOKEN"
    eval "local target_id=\$${other_upper}_ID"
    
    echo -e "\n${GREEN}Creating chat between ${creator_name} and ${other_name}...${NC}"
    
    local chat_response
    chat_response=$(curl -s -X POST "$BASE_URL/api/chats" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"direct\",\"user_ids\":[\"$target_id\"]}")
    
    CHAT_ID=$(parse_json "$chat_response" "chat.id")
    [ -z "$CHAT_ID" ] && CHAT_ID=$(echo "$chat_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    echo "$chat_response" | (jq . 2>/dev/null || cat)
    echo -e "${GREEN}  ✓ Chat ID: $CHAT_ID${NC}"
}

# Function to send message
send_message() {
    local sender_name=$1
    local receiver_name=$2
    local sender_upper=$(echo "$sender_name" | tr '[:lower:]' '[:upper:]')
    eval "local token=\$${sender_upper}_TOKEN"
    
    echo -e "\n${GREEN}${sender_name} sending message...${NC}"
    
    local response
    response=$(curl -s -X POST "$BASE_URL/api/messages" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{\"chat_id\":\"$CHAT_ID\",\"content\":\"Hello ${receiver_name}!\",\"content_type\":\"text\"}")
    
    MESSAGE_ID=$(parse_json "$response" "message.id")
    echo "$response" | (jq . 2>/dev/null || cat)
}

# --- Execution ---

register_user "$USER1_NAME" "$USER1_PHONE"
register_user "$USER2_NAME" "$USER2_PHONE"

search_user "$USER1_NAME" "$USER2_NAME"
create_chat "$USER1_NAME" "$USER2_NAME"

if [ -n "$CHAT_ID" ]; then
    send_message "$USER1_NAME" "$USER2_NAME"
fi

# Summary and export
echo -e "\n${YELLOW}=== Test Summary ===${NC}"
eval "echo \"MANAV_ID: \$MANAV_ID\""
eval "echo \"SAMI_ID: \$SAMI_ID\""

cat > test_variables.env << EOF
export MANAV_TOKEN="$(eval echo \$MANAV_TOKEN)"
export MANAV_ID="$(eval echo \$MANAV_ID)"
export SAMI_TOKEN="$(eval echo \$SAMI_TOKEN)"
export SAMI_ID="$(eval echo \$SAMI_ID)"
export CHAT_ID="$CHAT_ID"
export MESSAGE_ID="$MESSAGE_ID"
EOF

echo -e "\n${GREEN}Tests complete. Variables saved to test_variables.env${NC}"
