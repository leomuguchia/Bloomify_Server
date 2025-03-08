#!/bin/bash
# This script updates providers from providers.txt to use the new structure with randomized values.
# It assumes each line in providers.txt is in the format: ID:Token
# The update endpoint is: /api/providers/update/:id
# Each provider's location_geo field will be updated to a random coordinate within approx 5 km of the base.
# The script also updates:
#  - "rating" to a random value between 3.0 and 5.0.
#  - "completed_bookings" to a random integer between 10 and 100.
#
# Adjust the ranges as needed for your testing.

BASE_URL="http://192.168.100.19:8080"
UPDATE_ENDPOINT="${BASE_URL}/api/providers/update"

# Base coordinates (center point)
BASE_LAT=37.78
BASE_LON=-122.42

# Maximum random offset in degrees (approx 5 km is about 0.045 degrees)
MAX_OFFSET=0.045

# Files
PROVIDERS_FILE="providers.txt"

if [ ! -f "$PROVIDERS_FILE" ]; then
  echo "Providers file $PROVIDERS_FILE does not exist. Please run your seed script first."
  exit 1
fi

echo "Updating providers with randomized location_geo, rating, and completed_bookings..."

# Function to generate a random offset between -MAX_OFFSET and +MAX_OFFSET.
rand_offset() {
  # Generate a random number between 0 and MAX_OFFSET, then randomly flip the sign.
  offset=$(awk -v max="$MAX_OFFSET" 'BEGIN { srand(); print rand()*max }')
  sign=$((RANDOM % 2))
  if [ $sign -eq 0 ]; then
    echo "-$offset"
  else
    echo "$offset"
  fi
}

# Function to generate a random float between MIN and MAX.
rand_float() {
  min="$1"
  max="$2"
  awk -v min="$min" -v max="$max" 'BEGIN { srand(); print min + rand()*(max-min) }'
}

# Function to generate a random integer between MIN and MAX.
rand_int() {
  min="$1"
  max="$2"
  echo $(( (RANDOM % (max - min + 1)) + min ))
}

while IFS=: read -r id token; do
  # Generate random offsets for latitude and longitude.
  offsetLat=$(rand_offset)
  offsetLon=$(rand_offset)

  # Calculate new coordinates.
  newLat=$(echo "$BASE_LAT + $offsetLat" | bc -l)
  newLon=$(echo "$BASE_LON + $offsetLon" | bc -l)

  # Generate a random rating between 3.0 and 5.0.
  newRating=$(rand_float 3.0 5.0)
  # Generate a random completed_bookings between 10 and 100.
  newCompleted=$(rand_int 10 100)

  echo "Updating provider $id with new coordinates: lat=$newLat, lon=$newLon, rating=$newRating, completed_bookings=$newCompleted"

  # Build the JSON payload.
  PAYLOAD=$(cat <<EOF
{
  "location": "Test City",
  "location_geo": {"type": "Point", "coordinates": [$newLon, $newLat]},
  "rating": $newRating,
  "completed_bookings": $newCompleted
}
EOF
)

  # Call the update API.
  RESPONSE=$(curl -s -X PUT "${UPDATE_ENDPOINT}/${id}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${token}" \
    -d "$PAYLOAD")
  echo "Response: $RESPONSE"
  sleep 1
done < "$PROVIDERS_FILE"

echo "All provider updates complete."
