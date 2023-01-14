# currency-rest-api
This API will allow you to interact with current exchange rates.

The user has the ability to create a pair of currencies.
The user has the ability to transfer currency from one pair to another.

During initialization, a worker (goroutine) is launched, which should be launched once an hour 
(it goes through all the records in the table and updates the ratios for them taken from the Internet).

API urls:

1. Create record
POST /api/currency
query:
{
    "currencyFrom": "USD",
    "currencyTo": "RUB"
}
response:
  {"status": "success"} or {"status": "fail"}


2. Converting a value from one currency to another
PUT /api/currency
query:
{
    "currencyFrom": "USD",
    "currencyTo": "RUB"
    "value":  1
}
response:
  {"status":"success"} or {"status": "fail"}
  

3. Aggregation of added currency pairs
GET  /api/currency

example of resposne:
[
   {
       "currency_from":"USD","currency_to":"RUB","well":64.182,"updated_at":"2022-12-22T15:16:34+03:00"},
       {"currency_from":"TRY","currency_to":"RUB","well":10.513,"updated_at":"2022-12-22T15:16:34+03:00"},
       {"currency_from":"RUB","currency_to":"TRY","well":0.095,"updated_at":"2022-12-25T23:36:33+03:00"
   }
]

  
