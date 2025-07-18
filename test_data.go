package main

import (
	"github.com/charmbracelet/bubbles/list"
	"math/rand"
	"time"
)

var (
	taskNames = []string{
		"Buy milk and eggs", "Finish Q3 report", "Go to the post office", "Paint the fence",
		"Book flights to Hawaii", "Get an oil change", "Pick up dry cleaning", "Prepare presentation for Monday",
		"Buy a birthday gift for Sarah", "Clean the garage", "Research new laptops", "Schedule dentist appointment",
	}
	taskDescriptions = []string{
		"Get 2% milk and a dozen large eggs.", "Finalize the data and submit it to management.",
		"Mail the package to Mom.", "Use the new white paint we bought last week.",
		"Look for deals on Hawaiian Airlines.", "Go to the usual place on Main St.",
		"The one on the corner of 1st and Elm.", "Covering the latest sales figures.",
		"She likes books and candles.", "Organize the tools and sweep the floor.",
		"Looking for something with a good battery life.", "It's been over a year.",
	}
	folderTitles = []string{
		"Groceries", "Work Projects", "Personal Errands", "Home Improvement", "Vacation Planning", "Car Maintenance",
	}
	folderDescriptions = []string{
		"Things to buy from the store.", "Tasks related to my job.",
		"Things I need to get done around town.", "Projects to improve the house.",
		"Planning for the upcoming trip.", "Stuff to do for the car.",
	}
)

// randomString selects a random string from a slice of strings.
func randomString(r *rand.Rand, s []string) string {
	return s[r.Intn(len(s))]
}

// generateRandomTask creates a new Task with human-readable data.
func generateRandomTask(r *rand.Rand) Task {
	return Task{
		name:        randomString(r, taskNames),
		description: randomString(r, taskDescriptions),
		completed:   r.Intn(2) == 0,
		dueDate:     time.Now().Add(time.Duration(r.Intn(72)-24) * time.Hour), // Can be overdue
	}
}

// generateRandomTaskFolder creates a new TaskFolder with human-readable data.
func generateRandomTaskFolder(r *rand.Rand, depth int, parent *TaskFolder) *TaskFolder {
	folder := &TaskFolder{
		title:    randomString(r, folderTitles),
		desc:     randomString(r, folderDescriptions),
		progress: r.Float64(),
		parent:   parent,
	}

	// Add child tasks
	numTasks := r.Intn(4) + 1 // 1 to 4 child tasks
	for i := 0; i < numTasks; i++ {
		folder.children_tasks = append(folder.children_tasks, generateRandomTask(r))
	}

	// Add child folders (and limit recursion depth)
	if depth > 0 {
		numFolders := r.Intn(3) // 0 to 2 child folders
		for i := 0; i < numFolders; i++ {
			childFolder := generateRandomTaskFolder(r, depth-1, folder)
			folder.children_task_folders = append(folder.children_task_folders, childFolder)
		}
	}

	// Calculate status based on children
	completedCount := 0
	overdueCount := 0
	for _, task := range folder.children_tasks {
		if task.completed {
			completedCount++
		}
		if !task.completed && task.dueDate.Before(time.Now()) {
			overdueCount++
		}
	}
	folder.status = status{
		completed: completedCount,
		total:     len(folder.children_tasks),
		overdue:   overdueCount,
	}

	return folder
}

// return_test_data generates a list of list.Items for testing.
func return_test_data() []list.Item {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var items []list.Item
	numRootFolders := r.Intn(3) + 3 // 3 to 5 root folders
	for i := 0; i < numRootFolders; i++ {
		items = append(items, generateRandomTaskFolder(r, 2, nil)) // Max depth of 2
	}
	return items
}
