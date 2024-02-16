### Simple shooping cart learning skenario real world


```
Mendapatkan semua produk:
curl http://localhost:8080/products

Menambahkan produk baru:
curl -X POST -H "Content-Type: application/json" -d '{"name":"New Product","price":100,"quantity":10}' http://localhost:8080/products

Mendapatkan keranjang belanja untuk pengguna tertentu:

curl http://localhost:8080/cart/1

Menambahkan barang ke keranjang belanja pengguna tertentu:

curl -X POST -H "Content-Type: application/json" -d '{"product_id":1,"quantity":2}' http://localhost:8080/cart/1

Menghapus keranjang belanja pengguna tertentu:

curl -X DELETE http://localhost:8080/cart/1

Menghapus beberapa barang dari keranjang belanja pengguna tertentu:

curl -X DELETE -H "Content-Type: application/json" -d '[{"product_id":1,"quantity":2},{"product_id":2,"quantity":1}]' http://localhost:8080/cart/1/items

```
