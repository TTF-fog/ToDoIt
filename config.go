package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"os"
)

func MarshalToFile(filename string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func (f *TaskFolder) DeepCopy() *TaskFolder {
	if f == nil {
		return nil
	}

	newF := &TaskFolder{
		Name:   f.Name,
		Desc:   f.Desc,
		Status: f.Status,
	}

	if f.ChildrenTasks != nil {
		newF.ChildrenTasks = make([]*Task, len(f.ChildrenTasks))
		for i, task := range f.ChildrenTasks {
			newF.ChildrenTasks[i] = task.deepCopy()
		}
	}

	if f.ChildrenTaskFolders != nil {
		newF.ChildrenTaskFolders = make([]*TaskFolder, len(f.ChildrenTaskFolders))
		for i, childFolder := range f.ChildrenTaskFolders {
			copiedChild := childFolder.DeepCopy()
			newF.ChildrenTaskFolders[i] = copiedChild
		}
	}

	return newF
}

func (t *Task) deepCopy() *Task {
	if t == nil {
		return nil
	}

	newTask := &Task{
		Name:      t.Name,
		Desc:      t.Desc,
		Completed: t.Completed,
		DueDate:   t.DueDate,
		Overdue:   t.Overdue,
	}
	return newTask
}

func loadIntoTaskFolder(path string) (*TaskFolder, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var Folder TaskFolder
	err = json.Unmarshal(f, &Folder)
	if err != nil {
		panic(err)
	}
	return &Folder, nil
}
func reconstructFolderFromJSON(Folder *TaskFolder) {
	for _, item := range Folder.ChildrenTaskFolders {
		item.Parent = Folder
		reconstructFolderFromJSON(item)
	}
	for range Folder.ChildrenTasks {
		reconstructTasksFromJSON(Folder)
	}
	if Folder.Parent != nil {
		Folder.Progress = progress.New()
		if Folder.Status.Total > 0 {
			Folder.Progress.SetPercent(float64(Folder.Status.Completed/Folder.Status.Total) * 100)
		}

	}

}
func reconstructTasksFromJSON(Folder *TaskFolder) {
	for _, item := range Folder.ChildrenTasks {
		item.ParentFolder = Folder
	}
}
