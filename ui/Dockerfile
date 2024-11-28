# Build stage
FROM node:18-alpine AS BUILDER

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm install

# Copy source files
COPY . .

# Build the app
RUN npm run build

# Serve stage
FROM nginx:alpine
COPY --from=BUILDER /app/build /usr/share/nginx/html
EXPOSE 80

# Add nginx config to proxy API requests
COPY nginx.conf /etc/nginx/conf.d/default.conf 