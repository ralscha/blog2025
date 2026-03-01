FROM mcr.microsoft.com/playwright:v1.58.2-noble

WORKDIR /app

COPY package.json ./
RUN npm install --omit=dev

COPY server.js ./

EXPOSE 3000
CMD ["npm", "start"]
