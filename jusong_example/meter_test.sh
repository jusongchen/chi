curl http://localhost:3333/
curl http://localhost:3333/meters
curl -X DELETE http://localhost:3333/meters/5
curl -X DELETE http://localhost:3333/meters/2
curl -X DELETE http://localhost:3333/meters/3
curl -X DELETE http://localhost:3333/meters/1
curl -X DELETE http://localhost:3333/meters/4
echo 'all deleted'
curl http://localhost:3333/meters
curl -X POST -d '{"id":"will-be-omitted","project":"awesomeness"}' http://localhost:3333/meters
curl -X POST -d '{"project":"No-id-example"}' http://localhost:3333/meters
curl -X POST -d '{"project":"ora-example","ora_conn":"user1","ora_password":"Your-PassWd"}' http://localhost:3333/meters
curl -X POST -d '{"project":"duration-example","ora_conn":"user1","ora_password":"Your-PassWd","duration":"1m"}' http://localhost:3333/meters
curl -X POST -d '{"project":"BAD-duration-example","ora_conn":"user1","ora_password":"Your-PassWd","duration":"D1393"}' http://localhost:3333/meters
curl http://localhost:3333/meters
echo curl http://localhost:3333/meters/1
