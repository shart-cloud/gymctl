# Optimized Node.js Dockerfile with proper layer caching
FROM node:20-alpine

WORKDIR /app

# Copy dependency manifest first (changes less frequently)
COPY package.json .

# Install dependencies (cached if package.json unchanged)
RUN npm install

# Copy source code last (changes most frequently)
COPY index.js .

EXPOSE 3000
CMD ["node", "index.js"]
