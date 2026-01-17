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

# User Configuration - Change these as needed
USER1_NAME="Manav"
USER1_PHONE="9575417548"

USER2_NAME="Sami"
USER2_PHONE="8120890722"

# Convert names to uppercase for variable names
USER1_UPPER=$(echo "$USER1_NAME" | tr '[:lower:]' '[:upper:]')
USER2_UPPER=$(echo "$USER2_NAME" | tr '[:lower:]' '[:upper:]')

# Arrays to store responses
declare -A TOKENS
declare -A USER_IDS
declare -A RESPONSES

# Global variables
CHAT_ID=""
MESSAGE_ID=""

# Check if jq is available, otherwise use Python
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
        echo "$json" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    # Handle nested keys
    keys = '$key'.split('.')
    result = data
    for k in keys:
        if isinstance(result, dict) and k in result:
            result = result[k]
        else:
            print('')
            sys.exit(0)
    print(result if result is not None else '')
except:
    print('')
" 2>/dev/null
    fi
}

# Function to extract multiple values from JSON (BASH 4.3+ compatible)
extract_json_values() {
    local json="$1"
    local token_var_name="$2"
    local id_var_name="$3"
    
    local token=$(parse_json "$json" "token")
    local user_id=$(parse_json "$json" "user.id")
    
    # Use indirect reference to set variables
    eval "$token_var_name=\"\$token\""
    eval "$id_var_name=\"\$user_id\""
}

# Function to register a user
register_user() {
    local name=$1
    local phone=$2
    local upper_name=$(echo "$name" | tr '[:lower:]' '[:upper:]')
    
    echo -e "${GREEN}Registering ${name}...${NC}"
    
    # Make the request
    local response
    response=$(curl -s -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$phone\",\"name\":\"$name\"}")
    echo "curl -s -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$phone\",\"name\":\"$name\"}""
    echo $response
    # Check if response contains error
    if echo "$response" | grep -q "error\|Error\|failed\|Failed"; then
        echo -e "${RED}Registration failed for ${name}${NC}"
        echo -e "Response: $response\n"
        return 1
    fi
    
    RESPONSES[$name]="$response"
    
    echo -e "${BLUE}Full Response:${NC}"
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$response" | jq . 2>/dev/null || echo "$response"
    else
        echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
    fi
    echo ""
    
    # Extract token and ID using proper JSON parsing
    local token user_id
    extract_json_values "$response" token user_id
    
    if [ -z "$token" ] || [ -z "$user_id" ]; then
        echo -e "${RED}Failed to parse response for ${name}${NC}"
        echo -e "Trying alternative parsing method..."
        
        # Alternative parsing method
        token=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        user_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        
        if [ -z "$token" ] || [ -z "$user_id" ]; then
            echo -e "${RED}Could not extract token or ID from response${NC}"
            return 1
        fi
    fi
    
    TOKENS[$name]="$token"
    USER_IDS[$name]="$user_id"
    
    # Also set uppercase variable
    eval "${upper_name}_TOKEN=\"$token\""
    eval "${upper_name}_ID=\"$user_id\""
    
    echo -e "  ${name} Token: ${token:0:30}..."
    echo -e "  ${name} ID: $user_id"
    echo $TOKENS[$name]
    return 0
}

# Function to search for user
search_user() {
    local searcher_name=$1
    local target_name=$2
    local searcher_token=${TOKENS[$searcher_name]}
    
    if [ -z "$searcher_token" ]; then
        echo -e "${RED}No token found for ${searcher_name}${NC}"
        return 1
    fi
    
    echo -e "\n${GREEN}${searcher_name} searching for ${target_name}...${NC}"
    
    # Make the request
    local search_response
    search_response=$(curl -s -X GET "$BASE_URL/api/users/search?q=${target_name}" \
        -H "Authorization: Bearer $searcher_token")
    
    echo "curl -s -X GET "$BASE_URL/api/users/search?q=${target_name}" \
        -H "Authorization: Bearer $searcher_token""
    echo $search_response
    
    RESPONSES["${searcher_name}_search_${target_name}"]="$search_response"
    
    echo -e "${BLUE}Search Response:${NC}"
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$search_response" | jq . 2>/dev/null || echo "$search_response"
    else
        echo "$search_response" | python3 -m json.tool 2>/dev/null || echo "$search_response"
    fi
}

# Function to create chat
create_chat() {
    local creator_name=$1
    local other_name=$2
    local creator_token=${TOKENS[$creator_name]}
    local other_id=${USER_IDS[$other_name]}
    
    if [ -z "$creator_token" ]; then
        echo -e "${RED}No token found for ${creator_name}${NC}"
        return 1
    fi
    
    if [ -z "$other_id" ]; then
        echo -e "${RED}No ID found for ${other_name}${NC}"
        return 1
    fi
    
    echo -e "\n${GREEN}Creating chat between ${creator_name} and ${other_name}...${NC}"
    
    # Make the request
    local chat_response
    chat_response=$(curl -s -X POST "$BASE_URL/api/chats" \
        -H "Authorization: Bearer $creator_token" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"direct\",\"user_ids\":[\"$other_id\"]}")
    
    echo "curl -s -X POST "$BASE_URL/api/chats" \
        -H "Authorization: Bearer $creator_token" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"direct\",\"user_ids\":[\"$other_id\"]}""
    echo $chat_response

    RESPONSES["chat_${creator_name}_${other_name}"]="$chat_response"
    
    echo -e "${BLUE}Chat Response:${NC}"
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$chat_response" | jq . 2>/dev/null || echo "$chat_response"
    else
        echo "$chat_response" | python3 -m json.tool 2>/dev/null || echo "$chat_response"
    fi
    
    # Extract chat ID
    CHAT_ID=$(parse_json "$chat_response" "chat.id")
    
    if [ -n "$CHAT_ID" ]; then
        echo -e "\n${GREEN}  ✓ Chat created: $CHAT_ID${NC}"
    else
        # Try alternative parsing
        CHAT_ID=$(echo "$chat_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$CHAT_ID" ]; then
            echo -e "\n${GREEN}  ✓ Chat created: $CHAT_ID${NC}"
        else
            echo -e "${RED}  ✗ Failed to extract chat ID${NC}"
        fi
    fi
}

# Function to send message
send_message() {
    local sender_name=$1
    local receiver_name=$2
    local sender_token=${TOKENS[$sender_name]}
    
    if [ -z "$sender_token" ]; then
        echo -e "${RED}No token found for ${sender_name}${NC}"
        return 1
    fi
    
    if [ -z "$CHAT_ID" ]; then
        echo -e "${RED}No chat ID available${NC}"
        return 1
    fi
    
    echo -e "\n${GREEN}${sender_name} sending message to ${receiver_name}...${NC}"
    
    # Make the request
    local message_response
    message_response=$(curl -s -X POST "$BASE_URL/api/messages" \
        -H "Authorization: Bearer $sender_token" \
        -H "Content-Type: application/json" \
        -d "{\"chat_id\":\"$CHAT_ID\",\"content\":\"Hello ${receiver_name}!\",\"content_type\":\"text\"}")
    
    echo "curl -s -X POST "$BASE_URL/api/messages" \
        -H "Authorization: Bearer $sender_token" \
        -H "Content-Type: application/json" \
        -d "{\"chat_id\":\"$CHAT_ID\",\"content\":\"Hello ${receiver_name}!\",\"content_type\":\"text\"}""
    echo $message_response

    RESPONSES["message_${sender_name}_to_${receiver_name}"]="$message_response"
    
    echo -e "${BLUE}Message Response:${NC}"
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$message_response" | jq . 2>/dev/null || echo "$message_response"
    else
        echo "$message_response" | python3 -m json.tool 2>/dev/null || echo "$message_response"
    fi
    
    # Extract message ID
    MESSAGE_ID=$(parse_json "$message_response" "message.id")
    
    if [ -n "$MESSAGE_ID" ]; then
        echo -e "\n${GREEN}  ✓ Message sent: $MESSAGE_ID${NC}"
    else
        # Try alternative parsing
        MESSAGE_ID=$(echo "$message_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -n "$MESSAGE_ID" ]; then
            echo -e "\n${GREEN}  ✓ Message sent: $MESSAGE_ID${NC}"
        else
            echo -e "${YELLOW}  ⚠ Could not extract message ID${NC}"
        fi
    fi
}

# Function to get messages
get_messages() {
    local user_name=$1
    local user_token=${TOKENS[$user_name]}
    
    if [ -z "$user_token" ]; then
        echo -e "${RED}No token found for ${user_name}${NC}"
        return 1
    fi
    
    if [ -z "$CHAT_ID" ]; then
        echo -e "${RED}No chat ID available${NC}"
        return 1
    fi
    
    echo -e "\n${GREEN}${user_name} getting messages...${NC}"
    
    # Make the request
    local messages_response
    messages_response=$(curl -s -X GET "$BASE_URL/api/messages?chat_id=$CHAT_ID&offset=0&limit=50" \
        -H "Authorization: Bearer $user_token")
    
    echo "curl -s -X GET "$BASE_URL/api/messages?chat_id=$CHAT_ID&offset=0&limit=50" \
        -H "Authorization: Bearer $user_token""
    echo $messages_response

    RESPONSES["messages_${user_name}"]="$messages_response"
    
    echo -e "${BLUE}Messages Response:${NC}"
    if [ "$JSON_PARSER" = "jq" ]; then
        echo "$messages_response" | jq . 2>/dev/null || echo "$messages_response"
    else
        echo "$messages_response" | python3 -m json.tool 2>/dev/null || echo "$messages_response"
    fi
}

# Function to update message status
update_message_status() {
    local user_name=$1
    local status=$2
    local user_token=${TOKENS[$user_name]}
    
    if [ -z "$user_token" ]; then
        echo -e "${RED}No token found for ${user_name}${NC}"
        return 1
    fi
    
    if [ -z "$MESSAGE_ID" ]; then
        echo -e "${YELLOW}No message ID available - skipping status update${NC}"
        return 0
    fi
    
    echo -e "\n${GREEN}${user_name} marking message as ${status}...${NC}"
    
    # Make the request
    local status_response
    status_response=$(curl -s -X POST "$BASE_URL/api/messages/status" \
        -H "Authorization: Bearer $user_token" \
        -H "Content-Type: application/json" \
        -d "{\"message_id\":\"$MESSAGE_ID\",\"status\":\"$status\"}")
    
    echo "curl -s -X POST "$BASE_URL/api/messages/status" \
        -H "Authorization: Bearer $user_token" \
        -H "Content-Type: application/json" \
        -d "{\"message_id\":\"$MESSAGE_ID\",\"status\":\"$status\"}""
    echo $status_response

    RESPONSES["status_${user_name}_${status}"]="$status_response"
    
    echo -e "${BLUE}Status Update Response:${NC}"
    echo "$status_response"
}

# Function to test WebSocket connection
test_websocket() {
    local user_name=$1
    local user_token=${TOKENS[$user_name]}
    
    if [ -z "$user_token" ]; then
        echo -e "${RED}No token found for ${user_name}${NC}"
        return 1
    fi
    
    echo -e "\n${GREEN}Testing WebSocket connection for ${user_name}...${NC}"
    echo -e "${YELLOW}Note: This requires wscat to be installed${NC}"
    echo -e "Run manually: wscat -c \"ws://localhost:8080/ws?token=$user_token\""
}

# Main test execution
echo -e "${YELLOW}Testing with users:${NC}"
echo -e "  User 1: $USER1_NAME ($USER1_PHONE)"
echo -e "  User 2: $USER2_NAME ($USER2_PHONE)"
echo -e "  JSON Parser: $JSON_PARSER\n"

# Register users
echo -e "${YELLOW}=== Registration Phase ===${NC}"
register_user "$USER1_NAME" "$USER1_PHONE" || exit 1
register_user "$USER2_NAME" "$USER2_PHONE" || exit 1

echo "BOTH TOKENS: "$TOKENS


# Search for user
echo -e "\n${YELLOW}=== Search Phase ===${NC}"
search_user "$USER1_NAME" "$USER2_NAME"

# Create chat
echo -e "\n${YELLOW}=== Chat Creation Phase ===${NC}"
create_chat "$USER1_NAME" "$USER2_NAME"

# Send message if chat was created
if [ -n "$CHAT_ID" ]; then
    echo -e "\n${YELLOW}=== Messaging Phase ===${NC}"
    send_message "$USER1_NAME" "$USER2_NAME"
    
    # Get messages if message was sent
    if [ -n "$MESSAGE_ID" ]; then
        get_messages "$USER2_NAME"
        update_message_status "$USER2_NAME" "read"
        get_messages "$USER1_NAME"  # Check that Paa can also see the messages
    fi
fi

# Test WebSocket
echo -e "\n${YELLOW}=== WebSocket Test ===${NC}"
test_websocket "$USER1_NAME"
test_websocket "$USER2_NAME"

# Display summary
echo -e "\n${YELLOW}=== Test Summary ===${NC}"
echo -e "${GREEN}✓ Variables set:${NC}"
echo -e "  ${USER1_UPPER}_TOKEN: ${TOKENS[$USER1_NAME]:0:30}..."
echo -e "  ${USER1_UPPER}_ID: ${USER_IDS[$USER1_NAME]}"
echo -e "  ${USER2_UPPER}_TOKEN: ${TOKENS[$USER2_NAME]:0:30}..."
echo -e "  ${USER2_UPPER}_ID: ${USER_IDS[$USER2_NAME]}"

if [ -n "$CHAT_ID" ]; then
    echo -e "${GREEN}✓ Chat created:${NC} $CHAT_ID"
else
    echo -e "${RED}✗ No chat created${NC}"
fi

if [ -n "$MESSAGE_ID" ]; then
    echo -e "${GREEN}✓ Message sent:${NC} $MESSAGE_ID"
else
    echo -e "${YELLOW}⚠ No message sent${NC}"
fi

# Save variables to file for later use
cat > test_variables.env << EOF
# ChitChat Test Variables
# Generated on $(date)
# Users: $USER1_NAME ($USER1_PHONE) and $USER2_NAME ($USER2_PHONE)

export ${USER1_UPPER}_TOKEN="${TOKENS[$USER1_NAME]}"
export ${USER1_UPPER}_ID="${USER_IDS[$USER1_NAME]}"
export ${USER2_UPPER}_TOKEN="${TOKENS[$USER2_NAME]}"
export ${USER2_UPPER}_ID="${USER_IDS[$USER2_NAME]}"
EOF

if [ -n "$CHAT_ID" ]; then
    echo "export CHAT_ID=\"$CHAT_ID\"" >> test_variables.env
fi

if [ -n "$MESSAGE_ID" ]; then
    echo "export MESSAGE_ID=\"$MESSAGE_ID\"" >> test_variables.env
fi

# Add alias for easy sourcing
cat >> test_variables.env << EOF

# Alias to load all variables
load_test_vars() {
    source test_variables.env
    echo "Loaded test variables:"
    echo "  ${USER1_UPPER}_TOKEN: \${${USER1_UPPER}_TOKEN:0:30}..."
    echo "  ${USER2_UPPER}_TOKEN: \${${USER2_UPPER}_TOKEN:0:30}..."
    [ -n "\$CHAT_ID" ] && echo "  CHAT_ID: \$CHAT_ID"
    [ -n "\$MESSAGE_ID" ] && echo "  MESSAGE_ID: \$MESSAGE_ID"
}
EOF

echo -e "\n${GREEN}✓ Variables saved to test_variables.env${NC}"
echo -e "To use them: ${BLUE}source test_variables.env${NC}"
echo -e "Or use the helper: ${BLUE}load_test_vars${NC}"

# Display API endpoints for manual testing
echo -e "\n${YELLOW}=== Manual Testing Endpoints ===${NC}"
echo -e "${BLUE}For ${USER1_NAME}:${NC}"
echo -e "  Search: curl -X GET \"$BASE_URL/api/users/search?q=$USER2_NAME\" -H \"Authorization: Bearer \$${USER1_UPPER}_TOKEN\""
echo -e "  Chats: curl -X GET \"$BASE_URL/api/chats\" -H \"Authorization: Bearer \$${USER1_UPPER}_TOKEN\""
echo -e "${BLUE}For ${USER2_NAME}:${NC}"
echo -e "  Search: curl -X GET \"$BASE_URL/api/users/search?q=$USER1_NAME\" -H \"Authorization: Bearer \$${USER2_UPPER}_TOKEN\""
echo -e "  Chats: curl -X GET \"$BASE_URL/api/chats\" -H \"Authorization: Bearer \$${USER2_UPPER}_TOKEN\""

if [ -n "$CHAT_ID" ]; then
    echo -e "${BLUE}For chat $CHAT_ID:${NC}"
    echo -e "  Messages: curl -X GET \"$BASE_URL/api/messages?chat_id=$CHAT_ID\" -H \"Authorization: Bearer \$${USER1_UPPER}_TOKEN\""
    echo -e "  Chat details: curl -X GET \"$BASE_URL/api/chats/$CHAT_ID\" -H \"Authorization: Bearer \$${USER1_UPPER}_TOKEN\""
fi

echo -e "\n${GREEN}=== All tests completed! ===${NC}"
