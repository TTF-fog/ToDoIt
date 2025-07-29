package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
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

func randomString(r *rand.Rand, s []string) string {
	return s[r.Intn(len(s))]
}

func generateRandomTask(r *rand.Rand, completionChance float64, folder *TaskFolder) *Task {
	dueDate := time.Now().Add(time.Duration(r.Intn(72)-24) * time.Hour)
	completed := r.Float64() < completionChance
	return &Task{
		Name:         randomString(r, taskNames),
		Desc:         randomString(r, taskDescriptions),
		Completed:    completed,
		DueDate:      dueDate,
		ParentFolder: folder,
		Overdue:      !completed && dueDate.Before(time.Now()),
	}
}

func generateRandomTaskFolder(r *rand.Rand, depth int, parent *TaskFolder) *TaskFolder {
	m := progress.New(progress.WithDefaultGradient())

	folder := &TaskFolder{
		Name:     randomString(r, folderTitles),
		Desc:     randomString(r, folderDescriptions),
		Progress: m,
		Parent:   parent,
	}

	completionChance := r.Float64()
	numTasks := r.Intn(4) + 1
	for i := 0; i < numTasks; i++ {
		task := generateRandomTask(r, completionChance, folder)
		folder.ChildrenTasks = append(folder.ChildrenTasks, task)

	}

	if depth > 0 {
		numFolders := r.Intn(3)
		for i := 0; i < numFolders; i++ {
			childFolder := generateRandomTaskFolder(r, depth-1, folder)
			folder.ChildrenTaskFolders = append(folder.ChildrenTaskFolders, childFolder)
		}
	}

	completedCount := 0
	overdueCount := 0
	for _, task := range folder.ChildrenTasks {
		if task.Completed {
			completedCount++
		}
		if task.Overdue {
			overdueCount++
		}
	}
	folder.Status = status{
		completed: completedCount,
		total:     len(folder.ChildrenTasks),
		overdue:   overdueCount,
	}

	return folder
}

func return_test_data() []list.Item {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var items []list.Item
	numRootFolders := r.Intn(3) + 3
	for i := 0; i < numRootFolders; i++ {
		items = append(items, generateRandomTaskFolder(r, 2, nil))
	}
	return items
}
