package routes

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/karthiknayak6/url-shortner/database"
	"github.com/karthiknayak6/url-shortner/helpers"
)

type request struct {
	URL 			string			`json:"url"` 
	CustomShort 	string			`json:"short"`
	Expiry 			time.Duration	`json:"expiry"`
}

type response struct {
	URL 			string			`json:"url"` 
	CustomShort 	string			`json:"short"`
	Expiry 			time.Duration	`json:"expiry"`
	XRateRemaining	int				`json:"rate_limit"`
	XRateLimitReset	time.Duration	`json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)
	// if err := c.BodyParser(&body); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "cannot parse JSON",
	// 	})
	// }

	body.URL = c.FormValue("url")
	body.CustomShort = c.FormValue("short")
	

	//implement rate limiting
	r2 := database.CreateClient(1)

	defer r2.Close()

	_, err := r2.Get(database.Ctx, c.IP()).Result()

	fmt.Println(err)

	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ := r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Rate limit exceeded!!", "rate_limit_reset": limit / time.Nanosecond / time.Minute })
		}
	}

	

	
	// check if the input is an actual URL
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL",
		})
	}
	

	// check for the domain error
	// users may abuse the shortener by shorting the domain `localhost:3000` itself
	// leading to a inifite loop, so don't accept the domain for shortening
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "haha... nice try",
		})
	}

	// enforce https
	// all url will be converted to https before storing in database
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	val, _ := r.Get(database.Ctx, id).Result()
	
	if val != "" {
		errStr := `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
        <strong class="font-bold">Error!</strong>
        <span class="block sm:inline">Provided custom String is already in use</span>
        <span class="absolute top-0 bottom-0 right-0 px-4 py-3">
            
    	</div>`
		return c.SendString(errStr)
	
		// return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		// 	"error" : "URL short already in use",
		// })
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map {
			"error": "Unable to connect to server",
		})
		
	}
	
	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}
	r2.Decr(database.Ctx, c.IP())
	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)
	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id
	// return c.Status(fiber.StatusOK).JSON(resp)

	
	htmlString := fmt.Sprintf(`<ul class="flex h-48 justify-center flex-col text-xl space-y-2 ml-20">
	<li>
	  <span class="text-orange-500 font-bold">Shortened URL: </span>
	  <a class="text-blue-600" id="link" href="%v"
		>%v</a
	  >
	</li>

	<li><span class="text-orange-500 font-bold">Expiry:</span> %vhrs</li>
	<li>
	  <span class="text-orange-500 font-bold">Available requests:</span> %v
	</li>
	<li>
	  <span class="text-orange-500 font-bold"
		>Available requests reset time (10):</span
	  >
	  %v mins
	</li>
  </ul>`, resp.CustomShort,resp.CustomShort, int(resp.Expiry), resp.XRateRemaining, int(resp.XRateLimitReset))

	c.SendString(htmlString)
	return nil


}
