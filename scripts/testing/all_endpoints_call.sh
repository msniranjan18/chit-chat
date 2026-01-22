#!/bin/bash

# Colors for scannable output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${YELLOW}=== ChitChat Full API Suite Test ===${NC}\n"

BASE_URL="http://localhost:8080"
JSON_PARSER=$(command -v jq >/dev/null 2>&1 && echo "jq" || echo "python")

# Test Users
U1_NAME="Alice"; U1_PHONE="1111111111"
U2_NAME="Bob";   U2_PHONE="2222222222"
U3_NAME="Charlie"; U3_PHONE="3333333333"

# --- Helper Functions ---

call_api() {
    local method=$1
    local path=$2
    local token=$3
    local data=$4

    # All logs go to STDERR
    {
        echo -e "\n${YELLOW}>>> REQUEST${NC}"
        echo "METHOD : $method"
        echo "URL    : $BASE_URL$path"
        [ -n "$token" ] && echo "TOKEN  : Bearer $token"
        [ -n "$data" ] && echo "BODY   : $data"
    } >&2

    if [ "$method" == "GET" ] || [ "$method" == "DELETE" ]; then
        resp=$(curl -s -L -w "\nHTTP_STATUS:%{http_code}" \
            -X "$method" "$BASE_URL$path" \
            -H "Authorization: Bearer $token")
    else
        resp=$(curl -s -L -w "\nHTTP_STATUS:%{http_code}" \
            -X "$method" "$BASE_URL$path" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi

    body=$(echo "$resp" | sed -e '/HTTP_STATUS:/d')
    status=$(echo "$resp" | tr -d '\r' | sed -n 's/.*HTTP_STATUS://p')

    {
        echo -e "${BLUE}<<< RESPONSE${NC}"
        echo "STATUS : $status"
        echo "BODY   :"
        echo "$body"
        echo -e "${BLUE}-----------------------------${NC}"
    } >&2

    # Only JSON body goes to STDOUT
    echo "$body"
    echo ""
    echo ""
    sleep 5
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

    if [ -z "$token" ] || [ -z "$id" ]; then
        echo -e "${RED}Registration failed. Response was:${NC}"
        echo "$resp"
        exit 1
    fi
    eval "${prefix}_TOKEN=\"$token\""
    eval "${prefix}_ID=\"$id\""
    echo -e "${GREEN}Done (ID: $id)${NC}"
done

echo -e "\n${BLUE}Auth: Verify token${NC}"
call_api "GET" "/api/auth/verify" "$U1_TOKEN"

echo -e "\n${BLUE}Auth: Login${NC}"
login_resp=$(call_api "POST" "/api/auth/login" "" "{\"phone\":\"$U1_PHONE\"}")
NEW_TOKEN=$(parse_val "$login_resp" "token")

echo -e "\n${BLUE}Auth: Login${NC}"
login_resp=$(call_api "POST" "/api/auth/login" "" "{\"phone\":\"$U1_PHONE\"}")
NEW_TOKEN=$(parse_val "$login_resp" "token")


echo -e "\n${BLUE}Phase 2: User Profiles & Search${NC}"
echo "Updating Alice's profile..."
call_api "PUT" "/api/users/me" "$U1_TOKEN" "{\"status\":\"Feeling coding-tastic\"}" 

echo -e "\n${BLUE}Users: Get current user${NC}"
call_api "GET" "/api/users/me" "$U1_TOKEN"

echo -e "\n${BLUE}Users: Patch current user${NC}"
call_api "PATCH" "/api/users/me" "$U1_TOKEN" '{"status":"Updated via PATCH"}'

echo -e "\n${BLUE}Users: Get user by ID${NC}"
call_api "GET" "/api/users/$U2_ID" "$U1_TOKEN"

echo -e "\n${BLUE}Users: Online users${NC}"
call_api "GET" "/api/users/online" "$U1_TOKEN"

echo -e "\n${BLUE}Users: Sessions${NC}"
call_api "GET" "/api/users/sessions" "$U1_TOKEN"

echo "Searching for Bob..."
search_resp=$(call_api "GET" "/api/users/search?q=Bob" "$U1_TOKEN")
[ "$(parse_val "$search_resp" "[0].name")" == "Bob" ] && echo -e "  ${GREEN}✓ Search success${NC}"


echo -e "\n${BLUE}Phase 3: Contacts Management${NC}"
echo "Alice adding Bob to contacts..."
call_api "POST" "/api/contacts" "$U1_TOKEN" "{\"user_id\":\"$U2_ID\"}" 

echo "Alice adding Charlie to contacts..."
call_api "POST" "/api/contacts" "$U1_TOKEN" "{\"user_id\":\"$U3_ID\"}" 

echo -e "\n${BLUE}Contacts: Get all${NC}"
contacts=$(call_api "GET" "/api/contacts" "$U1_TOKEN")
CONTACT_ID=$(parse_val "$contacts" "[0].id")

echo -e "\n${BLUE}Contacts: Delete contact${NC}"
call_api "DELETE" "/api/contacts/$CONTACT_ID" "$U1_TOKEN"


echo -e "\n${BLUE}Chats: List chats${NC}"
call_api "GET" "/api/chats" "$U1_TOKEN"

echo -e "\n${BLUE}Chats: Search chats${NC}"
call_api "GET" "/api/chats/search?q=Gophers" "$U1_TOKEN"

echo -e "\n${BLUE}Phase 4: Group Chat & Messaging${NC}"
echo "Alice creating a group 'Gophers' with Bob and Charlie..."
group_resp=$(call_api "POST" "/api/chats" "$U1_TOKEN" "{\"type\":\"group\",\"name\":\"Gophers\",\"user_ids\":[\"$U2_ID\",\"$U3_ID\"]}")
GROUP_ID=$(parse_val "$group_resp" "chat.id")
[ -z "$GROUP_ID" ] && { echo "GROUP_ID missing"; exit 1; }
echo -e "  ${GREEN}✓ Group Created: $GROUP_ID${NC}"

echo -e "\n${BLUE}Chats: Get chat${NC}"
call_api "GET" "/api/chats/$GROUP_ID" "$U1_TOKEN"

echo -e "\n${BLUE}Chats: Update chat (PUT)${NC}"
call_api "PUT" "/api/chats/$GROUP_ID" "$U1_TOKEN" '{"name":"Gophers Updated"}'

echo -e "\n${BLUE}Chats: Update chat (PATCH)${NC}"
call_api "PATCH" "/api/chats/$GROUP_ID" "$U1_TOKEN" '{"name":"Gophers Patch"}'

echo -e "\n${BLUE}Chats: Get members${NC}"
call_api "GET" "/api/chats/$GROUP_ID/members" "$U1_TOKEN"

echo -e "\n${BLUE}Chats: Add member${NC}"
call_api "POST" "/api/chats/$GROUP_ID/members" "$U1_TOKEN" "{\"user_id\":\"$U3_ID\"}"

echo -e "\n${BLUE}Chats: Mark chat as read${NC}"
call_api "POST" "/api/chats/$GROUP_ID/read" "$U1_TOKEN" "{}"

echo "Alice sending message to group..."
msg_resp=$(call_api "POST" "/api/messages" "$U1_TOKEN" "{\"chat_id\":\"$GROUP_ID\",\"content\":\"Hello Gophers!\",\"content_type\":\"text\"}")
MSG_ID=$(parse_val "$msg_resp" "message.id")
[ -z "$MSG_ID" ] && { echo "MSG_ID missing"; exit 1; }

echo -e "\n${BLUE}Messages: Search messages${NC}"
call_api "GET" "/api/messages/search?q=Hello" "$U1_TOKEN"

echo -e "\n${BLUE}Messages: Update message (PUT)${NC}"
call_api "PUT" "/api/messages/$MSG_ID" "$U1_TOKEN" '{"content":"Edited message"}'

echo -e "\n${BLUE}Messages: Update message (PATCH)${NC}"
call_api "PATCH" "/api/messages/$MSG_ID" "$U1_TOKEN" '{"content":"Patched message"}'

echo -e "\n${BLUE}Messages: Delete message${NC}"
call_api "DELETE" "/api/messages/$MSG_ID" "$U1_TOKEN"


echo "Bob marking message as read..."
call_api "POST" "/api/messages/status" "$U2_TOKEN" "{\"message_id\":\"$MSG_ID\",\"status\":\"read\"}" 

echo "Charlie fetching group messages..."
msgs=$(call_api "GET" "/api/messages?chat_id=$GROUP_ID" "$U3_TOKEN")
echo -e "  ${GREEN}✓ Charlie received $(parse_val "$msgs" "messages | length") messages${NC}"

echo -e "\n${BLUE}Phase 5: Cleanup & Account${NC}"
echo "Alice leaving group..."
call_api "POST" "/api/chats/$GROUP_ID/leave" "$U1_TOKEN" "{}" 

echo "Alice logging out..."
call_api "POST" "/api/auth/logout" "$U1_TOKEN" "{}" 

echo -e "\n${YELLOW}=== Full Test Suite Completed ===${NC}"
