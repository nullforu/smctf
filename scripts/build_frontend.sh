#!/bin/bash

# Build frontend assets
cd frontend
npm ci
npm run build

if [ $? -ne 0 ]; then
  echo "Svelte build failed!"
  exit 1
fi

cd ..
