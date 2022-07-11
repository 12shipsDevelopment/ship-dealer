  ### Environment

- Ubuntu 20.04 x86_64 
- MySQL 8.0.26
- [Lotus v16.1.1](https://github.com/filecoin-project/lotus)

### How to run

1. ###### start lotus [lite-node](https://lotus.filecoin.io/lotus/install/lotus-lite/)

2. ###### create database

```
    $ mysql -uroot -hlocalhost -p -e "create database dealer_db";
```

3. ###### build dealer command [how to build](https://github.com/12shipsDevelopment/ship-dealer/blob/main/README.md)

4. ###### copy sample config files and change variables

```
   $ wget https://raw.githubusercontent.com/12shipsDevelopment/ship-dealer/main/docs/ship-deal.toml.sample -O ship-deal.toml
   $ vi ship-deal.toml
```

5. ###### run command

```
	$ ./ship-dealer
```


