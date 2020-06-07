package main

import (
	"fmt"
	"sync"

	"github.com/go-redis/redis/v7"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	key := "ticket_count"
	client.Set(key, "5", 0).Err()
	val, _ := client.Get(key).Result()
	fmt.Println("current ticket_count key val: ", val)

	getTicket(client, key)
}

func runTx(key string, id int) func(tx *redis.Tx) error {
	txf := func(tx *redis.Tx) error {
		n, err := tx.Get(key).Int()
		if err != nil && err != redis.Nil {
			return err
		}

		if n == 0 {
			fmt.Println(id, "票没了")
			return nil
		}

		// actual opperation (local in optimistic lock)
		n = n - 1

		// runs only if the watched keys remain unchanged
		_, err = tx.TxPipelined(func(pipe redis.Pipeliner) error {
			// pipe handles the error case
			pipe.Set(key, n, 0)
			return nil
		})
		return err
	}
	return txf
}

func getTicket(client *redis.Client, key string) {
	routineCount := 8
	var wg sync.WaitGroup
	wg.Add(routineCount)

	for i := 0; i < routineCount; i++ {
		go func(id int) {
			defer wg.Done()

			for {
				err := client.Watch(runTx(key, id), key)
				if err == nil {
					fmt.Println(id, "成功")
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
											