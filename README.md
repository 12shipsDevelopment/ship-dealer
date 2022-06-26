## Usage

1. clone project code

```bash
   $ git clone https://github.com/12shipsDevelopment/ship-dealer.git
   $ cd ship-dealer
```

2. Installation dependence.

```bash
   # install go
   $ wget https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
   $ rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.3.linux-amd64.tar.gz

   # build
   $ git clone https://github.com/filecoin-project/filecoin-ffi.git extern/filecoin-ffi 
   $ cd extern/filecoin-ffi && git checkout 7912389334e347bbb2eac0520c836830875c39de && ./install-filcrypto
   $ cd ../../
   $ go build
```
3. Running 
```bash
   # copy sample config file and change config, client need change [car] and [deal] filed, SP only need [market] field
   $ cp docs/ship-deal.toml.sample ship-deal.toml
   $ ./ship-deal
```

