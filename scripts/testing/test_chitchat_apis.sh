#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== ChitChat API Test ===${NC}\n"

# Base URL
BASE_URL="http://localhost:8080"

# User Configuration - Change these as needed
USER1_NAME="Paa"
USER1_PHONE="9005410039"

USER2_NAME="Maa"
USER2_PHONE="8765951530"

# Convert names to uppercase for variable names
USER1_UPPER=$(echo "$USER1_NAME" | tr '[:lower:]' '[:upper:]')
USER2_UPPER=$(echo "$USER2_NAME" | tr '[:lower:]' '[:upper:]')

# Arrays to store responses
declare -A TOKENS
declare -A USER_IDS
declare -A RESPONSES

# Function to register a user
register_user() {
    local name=$1
    local phone=$2
    local upper_name=$(echo "$name" | tr '[:lower:]' '[:upper:]')
    
    echo -e "${GREEN}Registering ${name}...${NC}"
    local response=$(curl -s -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone\":\"$phone\",\"name\":\"$name\"}")
    
    RESPONSES[$name]="$response"
    echo -e "Full Response: $response\n"
    
    # Extract token and ID
    local token=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    local user_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$token" ]; then
        echo -e "${RED}Failed to register ${name}${NC}"
        return 1
    fi
    
    TOKENS[$name]="$token"
    USER_IDS[$name]="$user_id"
    
    # Also set uppercase variable
    eval "${upper_name}_TOKEN=\"$token\""
    eval "${upper_name}_ID=\"$user_id\""
    
    echo -e "  ${name} Token: $token"
    echo -e "  ${name} ID: $user_id"
    return 0
}

# Function to search for user
search_user() {
    local searcher_name=$1
    local target_name=$2
    local searcher_token=${TOKENS[$searcher_name]}
    
    echo -e "\n${GREEN}${searcher_name} searching for ${target_name}...${NC}"
    local search_response=$(curl -s -X GET "$BASE_URL/api/users/search?q=${target_name}" \
        -H "Authorization: Bearer $searcher_token")
    
    RESPONSES["${searcher_name}_search_${target_name}"]="$search_response"
    echo -e "  Search Response: $search_response"
}

# Function to create chat
create_chat() {
    local creator_name=$1
    local other_name=$2
    local creator_token=${TOKENS[$creator_name]}
    local other_id=${USER_IDS[$other_name]}
    
    echo -e "\n${GREEN}Creating chat between ${creator_name} and ${other_name}...${NC}"
    local chat_response=$(curl -s -X POST "$BASE_URL/api/chats" \
        -H "Authorization: Bearer $creator_token" \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"direct\",\"user_ids\":[\"$other_id\"]}")
    
    RESPONSES["chat_${creator_name}_${other_name}"]="$chat_response"
    
    # Extract chat ID
    local chat_id=$(echo "$chat_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$chat_id" ]; then
        CHAT_ID="$chat_id"
        echo -e "  Chat ID: $CHAT_ID"
    else
        echo -e "${RED}  Failed to get chat ID${NC}"
        echo -e "  Response: $chat_response"
    fi
}

# Function to send message
send_message() {
    local sender_name=$1
    local receiver_name=$2
    local sender_token=${TOKENS[$sender_name]}
    
    echo -e "\n${GREEN}${sender_name} sending message to ${receiver_name}...${NC}"
    local message_response=$(curl -s -X POST "$BASE_URL/api/messages" \
        -H "Authorization: Bearer $sender_token" \
        -H "Content-Type: application/json" \
        -d "{\"chat_id\":\"$CHAT_ID\",\"content\":\"Hello ${receiver_name}!\",\"content_type\":\"text\"}")
    
    RESPONSES["message_${sender_name}_to_${receiver_name}"]="$message_response"
    
    # Extract message ID
    local message_id=$(echo "$message_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -n "$message_id" ]; then
        MESSAGE_ID="$message_id"
        echo -e "  Message ID: $MESSAGE_ID"
    else
        echo -e "${RED}  Failed to get message ID${NC}"
        echo -e "  Response: $message_response"
    fi
}

# Function to get messages
get_messages() {
    local user_name=$1
    local user_token=${TOKENS[$user_name]}
    
    echo -e "\n${GREEN}${user_name} getting messages...${NC}"
    local messages_response=$(curl -s -X GET "$BASE_URL/api/messages?chat_id=$CHAT_ID&offset=0&limit=50" \
        -H "Authorization: Bearer $user_token")
    
    RESPONSES["messages_${user_name}"]="$messages_response"
    
    # Try to extract message content
    local message_content=$(echo "$messages_response" | jq -r '.messages[0].content' 2>/dev/null)
    if [ $? -eq 0 ] && [ "$message_content" != "null" ]; then
        echo -e "  Last message: $message_content"
    else
        echo -e "  Messages response: $messages_response"
    fi
}

# Function to update message status
update_message_status() {
    local user_name=$1
    local status=$2
    local user_token=${TOKENS[$user_name]}
    
    echo -e "\n${GREEN}${user_name} marking message as ${status}...${NC}"
    local status_response=$(curl -s -X POST "$BASE_URL/api/messages/status" \
        -H "Authorization: Bearer $user_token" \
        -H "Content-Type: application/json" \
        -d "{\"message_id\":\"$MESSAGE_ID\",\"status\":\"$status\"}")
    
    RESPONSES["status_${user_name}_${status}"]="$status_response"
    echo -e "  Status Update: $status_response"
}

# Main test execution
echo -e "${YELLOW}Testing with users:${NC}"
echo -e "  User 1: $USER1_NAME ($USER1_PHONE)"
echo -e "  User 2: $USER2_NAME ($USER2_PHONE)\n"

# Register users
register_user "$USER1_NAME" "$USER1_PHONE" || exit 1
register_user "$USER2_NAME" "$USER2_PHONE" || exit 1

# Search for user (optional - may fail)
search_user "$USER1_NAME" "$USER2_NAME"

# Create chat
create_chat "$USER1_NAME" "$USER2_NAME"

# Send message if chat was created
if [ -n "$CHAT_ID" ]; then
    send_message "$USER1_NAME" "$USER2_NAME"
    
    # Get messages if message was sent
    if [ -n "$MESSAGE_ID" ]; then
        get_messages "$USER2_NAME"
        update_message_status "$USER2_NAME" "read"
    fi
fi

# Display summary
echo -e "\n${YELLOW}=== Test Summary ===${NC}"
echo -e "${GREEN}Variables set:${NC}"
echo -e "  ${USER1_UPPER}_TOKEN: ${TOKENS[$USER1_NAME]:0:30}..."
echo -e "  ${USER1_UPPER}_ID: ${USER_IDS[$USER1_NAME]}"
echo -e "  ${USER2_UPPER}_TOKEN: ${TOKENS[$USER2_NAME]:0:30}..."
echo -e "  ${USER2_UPPER}_ID: ${USER_IDS[$USER2_NAME]}"

if [ -n "$CHAT_ID" ]; then
    echo -e "  CHAT_ID: $CHAT_ID"
fi

if [ -n "$MESSAGE_ID" ]; then
    echo -e "  MESSAGE_ID: $MESSAGE_ID"
fi

# Save variables to file for later use
cat > test_variables.env << EOF
# ChitChat Test Variables
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

echo -e "\n${GREEN}Variables saved to test_variables.env${NC}"
echo -e "To use them: source test_variables.env"

echo -e "\n${GREEN}=== All tests completed! ===${NC}"
