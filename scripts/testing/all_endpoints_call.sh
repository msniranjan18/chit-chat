#!/bin/bash

# Colors for scannable output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${YELLOW}=== ChitChat Full API Suite Test ===${NC}\n"

BASE_URL="http://localhost:8080"
JSON_PARSER=$(command -v jq &> /dev/null && echo "jq" || echo "python")

# Test Users
U1_NAME="Alice"; U1_PHONE="1111111111"
U2_NAME="Bob";   U2_PHONE="2222222222"
U3_NAME="Charlie"; U3_PHONE="3333333333"

# --- Helper Functions ---

call_api() {
    local method=$1; local path=$2; local token=$3; local data=$4
    local auth_header=""
    [ -n "$token" ] && auth_header="-H \"Authorization: Bearer $token\""
    
    if [ "$method" == "GET" ] || [ "$method" == "DELETE" ]; then
        curl -s -X $method "$BASE_URL$path" -H "Authorization: Bearer $token"
    else
        curl -s -X $method "$BASE_URL$path" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

parse_val() {
    local json="$1"; local key="$2"
    if [ "$JSON_PARSER" == "jq" ]; then
        echo "$json" | jq -r ".$key" 2>/dev/null
    else
        python3 -c "import sys, json; d=json.load(sys.stdin); k='$key'.split('.'); r=d; [r := r.get(i) for i in k if isinstance(r, dict)]; print(r if r is not None else '')" <<< "$json"
    fi
}

# --- Test Phases ---

echo -e "${BLUE}Phase 1: Registration & Authentication${NC}"
for u in "U1:$U1_NAME:$U1_PHONE" "U2:$U2_NAME:$U2_PHONE" "U3:$U3_NAME:$U3_PHONE"; do
    IFS=":" read -r prefix name phone <<< "$u"
    echo -n "Registering $name... "
    resp=$(call_api "POST" "/api/auth/register" "" "{\"phone\":\"$phone\",\"name\":\"$name\"}")
    
    token=$(parse_val "$resp" "token")
    id=$(parse_val "$resp" "user.id")
    
    eval "${prefix}_TOKEN=\"$token\""
    eval "${prefix}_ID=\"$id\""
    echo -e "${GREEN}Done (ID: $id)${NC}"
done

echo -e "\n${BLUE}Phase 2: User Profiles & Search${NC}"
echo "Updating Alice's profile..."
call_api "PUT" "/api/users/me" "$U1_TOKEN" "{\"status\":\"Feeling coding-tastic\"}" > /dev/null

echo "Searching for Bob..."
search_resp=$(call_api "GET" "/api/users/search?q=Bob" "$U1_TOKEN")
[ "$(parse_val "$search_resp" "[0].name")" == "Bob" ] && echo -e "  ${GREEN}✓ Search success${NC}"

echo -e "\n${BLUE}Phase 3: Contacts Management${NC}"
echo "Alice adding Bob to contacts..."
call_api "POST" "/api/contacts" "$U1_TOKEN" "{\"contact_id\":\"$U2_ID\"}" > /dev/null
echo "Alice adding Charlie to contacts..."
call_api "POST" "/api/contacts" "$U1_TOKEN" "{\"contact_id\":\"$U3_ID\"}" > /dev/null

echo -e "\n${BLUE}Phase 4: Group Chat & Messaging${NC}"
echo "Alice creating a group 'Gophers' with Bob and Charlie..."
group_resp=$(call_api "POST" "/api/chats" "$U1_TOKEN" "{\"type\":\"group\",\"name\":\"Gophers\",\"user_ids\":[\"$U2_ID\",\"$U3_ID\"]}")
GROUP_ID=$(parse_val "$group_resp" "chat.id")
echo -e "  ${GREEN}✓ Group Created: $GROUP_ID${NC}"

echo "Alice sending message to group..."
msg_resp=$(call_api "POST" "/api/messages" "$U1_TOKEN" "{\"chat_id\":\"$GROUP_ID\",\"content\":\"Hello Gophers!\",\"content_type\":\"text\"}")
MSG_ID=$(parse_val "$msg_resp" "message.id")

echo "Bob marking message as read..."
call_api "POST" "/api/messages/status" "$U2_TOKEN" "{\"message_id\":\"$MSG_ID\",\"status\":\"read\"}" > /dev/null

echo "Charlie fetching group messages..."
msgs=$(call_api "GET" "/api/messages?chat_id=$GROUP_ID" "$U3_TOKEN")
echo -e "  ${GREEN}✓ Charlie received $(parse_val "$msgs" "messages | length") messages${NC}"

echo -e "\n${BLUE}Phase 5: Cleanup & Account${NC}"
echo "Alice leaving group..."
call_api "POST" "/api/chats/$GROUP_ID/leave" "$U1_TOKEN" "{}" > /dev/null

echo "Alice logging out..."
call_api "POST" "/api/auth/logout" "$U1_TOKEN" "{}" > /dev/null

echo -e "\n${YELLOW}=== Full Test Suite Completed ===${NC}"
