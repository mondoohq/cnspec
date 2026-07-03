FROM node:20-slim
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install && npm cache clean --force
