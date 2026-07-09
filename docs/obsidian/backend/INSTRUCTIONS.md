# Idea
The main idea of the backend, is to connect the frontend to the backend but with a diferent perspective, the backend must be built
in microservices, the main microservice must be the auth service, this service must also be a load balancer, because all the requests
that the frontend makes, must pass through this service, to check if the user is logged in, after the system checks that the user is
logged in, it will redirect the request to the correspondent service.

# Tech Stack
- Backend programming language: Go
- Database: Postgresql + Supabase

## Auth service
This auth service, wich is going to be the first one to be built, is gonna be built in the go programming language, and connect
directly to postgresql in supabase, the credentials are in the .env.local file in the auth service inside the microservices/ folder.
This service must receive the request from the frontend, so add a cors file, so i can add manually the addresses the backend can
accept, for now is gonna be localhost:3000, because the frontend is built in nextjs, it must have a correspondent middleware, jwt
sessions, because supabase manages auth with jwt, and also i want this to be able to be a load balancer with nginx and docker, every
container must have its nginx file so a new container can be set up, i know this must be from the docker-compose.yaml file, to start
make 5 containers so the loading can be distributed through all these containers, add logs in the terminal so i can see where the 
request are going to which container and address, in the auth service, create the signup and login logic, also create an endpoint to
get the basic information of a user, like name, email and token, so the frontend can know if the user is still logged in, this
service will get request so it must be able to redirect to the correspondient service, example.
Let's say a user makes a request to get the information of a serie, it is going to look like this:
- User presses button
- Frontend makes request GET /api/v1/serie/idker92038jdlairjlei392
- Load balancer checks request, but first checks if the user is logged in
- loggedIn ? redirect to GET /api/v1/serie/idker92038jdlairjlei392 : alert the frontend that the user is not logged in
This service must be in /microservices/auth
