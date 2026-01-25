# CTF Platform

```shell
git clone https://github.com/nullforu/smctf.git
cd smctf

cd frontend
npm install && npm run build
cd ..

go build -o smctf ./cmd/server
./smctf

# or: go run ./cmd/server
```

> 전공 동아리 [Null4U](https://github.com/nullforu)에서 Go 언어를 다루지도 않고, 아마 학교에서 Go 언어를 사용하는 인간이 저밖에 없지 않을까 생각합니다.
> 
> 그래도 Go 언어를 선택한 이유가 무엇이냐..?
> 
> - 기존에 자주 사용하던 NodeJS의 NestJS 프레임워크는 너무 무거웠음. (DI, 복잡한 구조와 런타임 데코레이터, 많은 빌트인 기능으로 인해 무겁고 운영 상 오버헤드가 있을 수 있었음)
> - 그렇다고 가벼운 ExpressJS 프레임워크는 너무 자유로워서 유지보수가 어렵다고 판단, Fastify도 고려했으나 익숙하지 않았음.
> - 백엔드 개발을 위한 언어/런타임 중 수준있게 다룰 수 있는 언어/런타임이 사실상 NodeJS와 Go 언어밖에 없었음. (Python, Ruby, Java 등은 개인적으로 선호하지 않음)
> - Go 언어는 컴파일링을 거치면 단일 바이너리로 배포 가능, (이론상) 빠름, 정적 타이핑, 쉬운 문법, 나름 생태계가 갖춰짐, 러닝 커브가 완만함.
>   - Go 언어를 처음 접했을 2019년 당시엔 Go 언어의 생태계가 부족하다고 판단하였으나, 현재는 어느정도 갖춰진 상태라고 판단하였음.
> - Gin, Fiber, Echo 등의 여러 웹 프레임워크가 있었으나 생태계가 가장 크고 안정적인 Gin 프레임워크를 선택함.
> - ORM도 여러 후보를 고려했었으나 최종적으로 Bun을 선택하였음.
> 
> \- 프로젝트의 유일 메인테이너이자 동아리 부장 [@yulmwu](https://github.com/yulmwu) \-
