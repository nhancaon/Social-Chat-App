FROM node:26.4.0-alpine AS build
WORKDIR /usr/src/app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install
COPY frontend .
RUN npm run build

FROM nginx:1.31.2-alpine
COPY --from=build /usr/src/app/dist /usr/share/nginx/html
COPY frontend/.nginx/nginx.conf /etc/nginx/nginx.conf
COPY frontend/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/docker-entrypoint.sh"]