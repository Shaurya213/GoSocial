#!/bin/bash
echo "🔍 GoSocial User Service - Complete API Discovery"
echo "================================================"

echo ""
echo "📋 All Available Services:"
grpcurl -plaintext localhost:8080 list

echo ""
echo "📋 UserService Methods:"
grpcurl -plaintext localhost:8080 list api.v1.UserService

echo ""
echo "🔍 Method Signatures:"
echo "===================="

METHODS=(
  "Register"
  "Login"
  "GetProfile"
  "UpdateProfile"
  "SendFriendRequest"
  "AcceptFriendRequest"
  "ListFriends"
  "RegisterDevice"
  "RemoveDevice"
  "GetUserDevices"
)

for method in "${METHODS[@]}"; do
  echo ""
  echo "📝 $method:"
  grpcurl -plaintext localhost:8080 describe api.v1.UserService.$method
done

echo ""
echo "🏗️ Message Structures:"
echo "======================"

MESSAGES=(
  "RegisterRequest"
  "LoginRequest"
  "AuthResponse"
  "GetProfileRequest"
  "ProfileResponse"
  "UpdateProfileRequest"
  "FriendRequest"
  "FriendAcceptRequest"
  "UserID"
  "Friend"
  "FriendList"
  "StatusResponse"
  "DeviceTokenRequest"
  "DeviceToken"
  "DeviceTokenList"
)

for message in "${MESSAGES[@]}"; do
  echo ""
  echo "📋 $message:"
  grpcurl -plaintext localhost:8080 describe api.v1.$message
done

echo ""
echo "✅ Discovery Complete!"
