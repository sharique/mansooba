module fixture.example/backend

go 1.27

require fixture.example/permdep v0.0.0

replace fixture.example/permdep => @FIXTURES@/fixtures-src/permdep
