#!/bin/bash 
set -e
#set -x
clear 

# List of files to display
files=("portscanner.go" "structs.go" "helpers.go" "email.go" "main_test.go")

# Loop through each file and display its name and contents
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "=== File: $file ==="
        cat "$file"
        echo -e "\n"
    else
        echo "=== File: $file ==="
        echo "File not found"
        echo -e "\n"
    fi
done