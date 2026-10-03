# ---- build ----
FROM node:22-alpine AS build

WORKDIR /app

# Las URLs de los backends se inyectan en build time (Vite las congela en el
# bundle). Se pasan desde docker-compose como build args.
ARG VITE_PAGOS_URL
ARG VITE_MOTOR_URL
ARG VITE_MOTOR_WS_URL
ENV VITE_PAGOS_URL=$VITE_PAGOS_URL \
    VITE_MOTOR_URL=$VITE_MOTOR_URL \
    VITE_MOTOR_WS_URL=$VITE_MOTOR_WS_URL

COPY package.json package-lock.json* ./
RUN npm install

COPY . .
RUN npm run build

# ---- runtime ----
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
