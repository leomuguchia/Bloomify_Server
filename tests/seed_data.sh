#!/bin/bash
# This script creates 20 providers with randomized geo-coordinates within ~5km of the base point.
# It uses a set of creative names and saves each provider's id and token in providers_output.txt.
# All providers are set with the necessary criteria for matching.
# Adjust values as needed.

BASE_URL="http://192.168.100.19:8080"
REGISTER_ENDPOINT="${BASE_URL}/api/providers/register"

# Base coordinates (center of search)
BASE_LAT=37.78
BASE_LON=-122.42

# Maximum random offset in degrees (~5 km is about 0.045 degrees)
MAX_OFFSET=0.045

# Output file for storing provider id and token
OUTPUT_FILE="providers_output.txt"
> "$OUTPUT_FILE"  # Clear file

# An array of creative names.
names=(
  "Kelvin Cleaning"
  "Michael's Cleaners"
  "Owen Services"
  "John's Cleaning"
  "Blessed Laundry"
  "Hope Dealers"
  "Creative Cleaners"
  "Dynamic Detergents"
  "Sparkle Solutions"
  "Pure Clean Co."
  "Fresh Start Cleaning"
  "Elite Cleaners"
  "NextGen Clean"
  "Prime Shine"
  "Urban Clean"
  "Sunrise Cleaning"
  "EcoClean"
  "Apex Services"
  "TopNotch Cleaners"
  "Bright Future Cleaners"
)

rand_offset() {
  awk -v max="$MAX_OFFSET" 'BEGIN { srand(); print (rand()*2-1)*max }'
}

for i in {0..19}; do
  # Pick a random name.
  idx=$((RANDOM % ${#names[@]}))
  name="${names[$idx]}"
  legal_name="${name} LLC"
  # Generate an email by lowercasing the name, removing spaces, and appending the index.
  email=$(echo "$name" | tr '[:upper:]' '[:lower:]' | tr -d ' ')"$i@example.com"
  
  offsetLat=$(rand_offset)
  offsetLon=$(rand_offset)
  newLat=$(echo "$BASE_LAT + $offsetLat" | bc -l)
  newLon=$(echo "$BASE_LON + $offsetLon" | bc -l)
  
  # Generate random rating and completed bookings.
  newRating=$(awk -v min=3.0 -v max=5.0 'BEGIN { srand(); print min + rand()*(max-min) }')
  newCompleted=$(( (RANDOM % 91) + 10 ))
  
  echo "Registering provider: '$name' ($email) with coordinates: [$newLon, $newLat], rating: $newRating, completed_bookings: $newCompleted"
  
  PAYLOAD=$(cat <<EOF
{
  "provider_name": "$name",
  "legal_name": "$legal_name",
  "email": "$email",
  "phone_number": "555-$(printf "%04d" $i)",
  "password": "password$i",
  "service_type": "cleaning",
  "location": "Test City",
  "location_geo": {"type": "Point", "coordinates": [$newLon, $newLat]},
  "kyp_document": "doc-ref-$i",
  "kyp_verification_code": "verify$i",
  "verification_status": "verified",
  "verification_level": "advanced",
  "insurance_docs": ["insurance1.pdf", "insurance2.pdf"],
  "tax_pin": "123456789",
  "advanced_verified": true,
  "status": "active"
}
EOF
)
  
  RESPONSE=$(curl -s -X POST "$REGISTER_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")
  
  echo "Response: $RESPONSE"
  
  # Extract id and token using jq and append to the output file.
  id=$(echo "$RESPONSE" | jq -r '.id')
  token=$(echo "$RESPONSE" | jq -r '.token')
  echo "$id:$token" >> "$OUTPUT_FILE"
  
  sleep 1
done

echo "All provider registrations complete. Saved output to $OUTPUT_FILE"
