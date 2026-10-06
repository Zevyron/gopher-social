package db

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/Zevyron/gopher-social/internal/store" // Adjust this if your module path differs
	"github.com/brianvoe/gofakeit/v7"
)

func Seed(store store.Storage) {
	ctx := context.Background()

	users := generateUsers(100)
	for _, user := range users {
		if err := store.Users.Create(ctx, nil, user); err != nil {
			log.Println("Error creating user:", err)
			return
		}
	}

	posts := generatePosts(200, users)
	for _, post := range posts {
		if err := store.Posts.Create(ctx, post); err != nil {
			log.Println("Error creating post:", err)
			return
		}
	}

	comments := generateComments(500, users, posts)
	for _, comment := range comments {
		if err := store.Comments.Create(ctx, comment); err != nil {
			log.Println("Error creating comment:", err)
			return
		}
	}

	log.Println("Seeding complete")
}

func generateUsers(num int) []*store.User {
	users := make([]*store.User, num)
	
	template := &store.User{}
	if err := template.Password.Set("1234567"); err != nil {
		log.Println("failed to hash seed password: ", err)
	}

	for i := 0; i < num; i++ {
		users[i] = &store.User{
			Username:  gofakeit.Username() + fmt.Sprintf("%d", i),
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
			Email:     gofakeit.Email(),
			Password:  template.Password, 
		}
	}

	return users
}

func generatePosts(num int, users []*store.User) []*store.Post {
	posts := make([]*store.Post, num)
	for i := 0; i < num; i++ {
		user := users[rand.Intn(len(users))]

		posts[i] = &store.Post{
			UserID:  user.ID,
			Title:   gofakeit.Sentence(4),
			Content: gofakeit.Paragraph(1, 3, 5, "\n"),
			Tags: []string{
				gofakeit.Word(),
				gofakeit.Word(),
			},
		}
	}

	return posts
}

func generateComments(num int, users []*store.User, posts []*store.Post) []*store.Comment {
	cms := make([]*store.Comment, num)
	for i := 0; i < num; i++ {
		cms[i] = &store.Comment{
			PostID:  posts[rand.Intn(len(posts))].ID,
			UserID:  users[rand.Intn(len(users))].ID,
			Content: gofakeit.Sentence(10),
		}
	}
	return cms
}