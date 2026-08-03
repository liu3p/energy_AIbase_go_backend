# localweb_backend

## Init submodule

```sh

git submodule update --recursive --init
```

## Update submodule

```sh
git submodule update --init
```

## Setting private registry

```sh
npm config set registry http://repo.sfere.local/repository/npm-group/
```

## Build binary file

```sh
go build -o ess-localweb .
```

---

整合前端的编译后，全部的编译流程已经写到build.sh中了。
build.sh中会删除原子目录，然后新建子目录拉取子模块（前端）的代码进行编译处理。
