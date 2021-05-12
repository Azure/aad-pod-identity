FROM openjdk:8-slim as builder

RUN mkdir -p /build
WORKDIR /build
COPY pom.xml /build
COPY mvnw /build
COPY .mvn /build/.mvn

RUN ./mvnw -B dependency:resolve dependency:resolve-plugins

COPY src /build/src

RUN ./mvnw -DskipTests package



FROM openjdk:8-slim

WORKDIR /app
COPY --from=builder /build/target/*.jar app.jar
ENTRYPOINT ["java","-jar","/app/app.jar"]
