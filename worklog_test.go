package main

import (
	"testing"
)

func strPtr(s string) *string {
	return &s
}

// Test helper to create a simple worklog with tasks for testing
func createTestWorklog() *Worklog {
	return &Worklog{
		path:  "testdata/test.ics",
		tasks: []*Task{},
	}
}

func TestFindTaskByUUID(t *testing.T) {
	worklog := createTestWorklog()

	// Create a hierarchy: root -> child -> grandchild
	root := NewTask("root")
	child := NewTask("child")
	grandchild := NewTask("grandchild")

	worklog.tasks = append(worklog.tasks, root)
	root.children = append(root.children, child)
	child.parent = root
	child.children = append(child.children, grandchild)
	grandchild.parent = child

	// Test finding at root level
	found := worklog.FindTaskByUUID(root.uuid)
	if found == nil || found.uuid != root.uuid {
		t.Errorf("Expected to find root task by UUID")
	}

	// Test finding in first level of children
	found = worklog.FindTaskByUUID(child.uuid)
	if found == nil || found.uuid != child.uuid {
		t.Errorf("Expected to find child task by UUID")
	}

	// Test finding in deeper level
	found = worklog.FindTaskByUUID(grandchild.uuid)
	if found == nil || found.uuid != grandchild.uuid {
		t.Errorf("Expected to find grandchild task by UUID")
	}

	// Test finding non-existent UUID
	found = worklog.FindTaskByUUID("non-existent-uuid")
	if found != nil {
		t.Errorf("Expected nil for non-existent UUID")
	}
}

func TestUpdateTaskUpdatesNameAndDescription(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Original Name")
	task.description = "Original Description"
	worklog.tasks = append(worklog.tasks, task)

	err := worklog.UpdateTask(task.uuid, TaskUpdate{
		Name:        strPtr("New Name"),
		Description: strPtr("New Description"),
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if task.name != "New Name" {
		t.Errorf("Expected name to be 'New Name', got '%s'", task.name)
	}

	if task.description != "New Description" {
		t.Errorf("Expected description to be 'New Description', got '%s'", task.description)
	}
}

func TestUpdateTaskMoveToNewParent(t *testing.T) {
	worklog := createTestWorklog()

	// Create: parent1 -> child, and separate parent2
	parent1 := NewTask("parent1")
	child := NewTask("child")
	parent2 := NewTask("parent2")

	worklog.tasks = append(worklog.tasks, parent1, parent2)
	parent1.children = append(parent1.children, child)
	child.parent = parent1

	// Move child from parent1 to parent2
	err := worklog.UpdateTask(child.uuid, TaskUpdate{
		ParentUUID: strPtr(parent2.uuid),
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify child is now under parent2
	if child.parent != parent2 {
		t.Errorf("Expected child's parent to be parent2")
	}

	// Verify parent2 has child
	if len(parent2.children) != 1 || parent2.children[0].uuid != child.uuid {
		t.Errorf("Expected parent2 to have child as its only child")
	}

	// Verify parent1 no longer has child
	if len(parent1.children) != 0 {
		t.Errorf("Expected parent1 to have no children")
	}
}

func TestUpdateTaskMoveToRoot(t *testing.T) {
	worklog := createTestWorklog()

	// Create: parent -> child
	parent := NewTask("parent")
	child := NewTask("child")

	worklog.tasks = append(worklog.tasks, parent)
	parent.children = append(parent.children, child)
	child.parent = parent

	// Move child to root (empty parentUUID means move to root)
	err := worklog.UpdateTask(child.uuid, TaskUpdate{
		ParentUUID: strPtr(""),
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Child should now be at root (no parent)
	if child.parent != nil {
		t.Errorf("Expected child to have no parent after moving to root")
	}

	// Child should be in worklog.tasks
	found := false
	for _, t := range worklog.tasks {
		if t.uuid == child.uuid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected child to be in root tasks after moving to root")
	}
}

func TestUpdateTaskSelfParenting(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("task")
	worklog.tasks = append(worklog.tasks, task)

	// Try to set task as its own parent
	err := worklog.UpdateTask(task.uuid, TaskUpdate{
		ParentUUID: strPtr(task.uuid),
	})
	if err == nil {
		t.Errorf("Expected error when setting task as its own parent")
	}

	// Verify the error message mentions cycle
	if err != nil && err.Error() != "cannot set task as its own parent (cycle detected)" {
		t.Errorf("Expected specific error message about self-parenting, got: %v", err)
	}

	// Verify task is still at root
	if task.parent != nil {
		t.Errorf("Expected task to remain without parent after failed update")
	}
}

func TestUpdateTaskMoveToDescendant(t *testing.T) {
	worklog := createTestWorklog()

	// Create: grandparent -> parent -> child
	grandparent := NewTask("grandparent")
	parent := NewTask("parent")
	child := NewTask("child")

	worklog.tasks = append(worklog.tasks, grandparent)
	grandparent.children = append(grandparent.children, parent)
	parent.parent = grandparent
	parent.children = append(parent.children, child)
	child.parent = parent

	// Try to move grandparent to be a child of its descendant (child)
	err := worklog.UpdateTask(grandparent.uuid, TaskUpdate{
		ParentUUID: strPtr(child.uuid),
	})
	if err == nil {
		t.Errorf("Expected error when moving task to its descendant")
	}

	// Verify the error message mentions cycle
	expectedMsg := "cannot move task to a descendant (cycle detected)"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedMsg, err)
	}

	// Verify tree structure is unchanged
	if grandparent.parent != nil {
		t.Errorf("Expected grandparent to remain at root")
	}
	if parent.parent != grandparent {
		t.Errorf("Expected parent's parent to remain grandparent")
	}
	if child.parent != parent {
		t.Errorf("Expected child's parent to remain parent")
	}
}

func TestUpdateTaskMoveToChild(t *testing.T) {
	worklog := createTestWorklog()

	// Create: parent -> child
	parent := NewTask("parent")
	child := NewTask("child")

	worklog.tasks = append(worklog.tasks, parent)
	parent.children = append(parent.children, child)
	child.parent = parent

	// Try to move parent to be a child of its child (immediate cycle)
	err := worklog.UpdateTask(parent.uuid, TaskUpdate{
		ParentUUID: strPtr(child.uuid),
	})
	if err == nil {
		t.Errorf("Expected error when moving parent to its child")
	}

	// Verify tree structure is unchanged
	if parent.parent != nil {
		t.Errorf("Expected parent to remain at root")
	}
	if child.parent != parent {
		t.Errorf("Expected child's parent to remain parent")
	}
}

func TestUpdateTaskNonExistentTask(t *testing.T) {
	worklog := createTestWorklog()

	err := worklog.UpdateTask("non-existent-uuid", TaskUpdate{
		Name: strPtr("New Name"),
	})
	if err == nil {
		t.Errorf("Expected error when updating non-existent task")
	}

	expectedMsg := "task with UUID 'non-existent-uuid' not found"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedMsg, err)
	}
}

func TestUpdateTaskNonExistentParent(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("task")
	worklog.tasks = append(worklog.tasks, task)

	err := worklog.UpdateTask(task.uuid, TaskUpdate{
		ParentUUID: strPtr("non-existent-parent"),
	})
	if err == nil {
		t.Errorf("Expected error when setting non-existent parent")
	}

	expectedMsg := "parent task with UUID 'non-existent-parent' not found"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedMsg, err)
	}
}

func TestUpdateTaskClearDescription(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("task")
	task.description = "Original Description"
	worklog.tasks = append(worklog.tasks, task)

	err := worklog.UpdateTask(task.uuid, TaskUpdate{
		Description: strPtr(""),
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if task.description != "" {
		t.Errorf("Expected description to be cleared, got '%s'", task.description)
	}
}

func TestUpdateTaskCannotClearName(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Original Name")
	worklog.tasks = append(worklog.tasks, task)

	err := worklog.UpdateTask(task.uuid, TaskUpdate{
		Name: strPtr(""),
	})
	if err == nil {
		t.Fatalf("Expected error when clearing task name")
	}

	expectedMsg := "cannot clear task name"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedMsg, err)
	}

	if task.name != "Original Name" {
		t.Errorf("Expected name to remain unchanged, got '%s'", task.name)
	}
}

func TestUpdateTaskNoChanges(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Original Name")
	task.description = "Original Description"
	worklog.tasks = append(worklog.tasks, task)

	// Empty update - no fields provided
	err := worklog.UpdateTask(task.uuid, TaskUpdate{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if task.name != "Original Name" {
		t.Errorf("Expected name to remain unchanged, got '%s'", task.name)
	}
	if task.description != "Original Description" {
		t.Errorf("Expected description to remain unchanged, got '%s'", task.description)
	}
}

func TestIsDescendantOf(t *testing.T) {
	// Create: grandparent -> parent -> child, and sibling (under parent)
	grandparent := NewTask("grandparent")
	parent := NewTask("parent")
	child := NewTask("child")
	sibling := NewTask("sibling")
	unrelated := NewTask("unrelated")

	grandparent.children = append(grandparent.children, parent)
	parent.parent = grandparent
	parent.children = append(parent.children, child, sibling)
	child.parent = parent
	sibling.parent = parent

	// Test: child is descendant of grandparent
	if !isDescendantOf(grandparent, child) {
		t.Errorf("Expected child to be descendant of grandparent")
	}

	// Test: child is descendant of parent
	if !isDescendantOf(parent, child) {
		t.Errorf("Expected child to be descendant of parent")
	}

	// Test: sibling is descendant of grandparent
	if !isDescendantOf(grandparent, sibling) {
		t.Errorf("Expected sibling to be descendant of grandparent")
	}

	// Test: grandparent is NOT descendant of child
	if isDescendantOf(child, grandparent) {
		t.Errorf("Expected grandparent NOT to be descendant of child")
	}

	// Test: parent is NOT descendant of child
	if isDescendantOf(child, parent) {
		t.Errorf("Expected parent NOT to be descendant of child")
	}

	// Test: sibling is NOT descendant of child
	if isDescendantOf(child, sibling) {
		t.Errorf("Expected sibling NOT to be descendant of child")
	}

	// Test: unrelated is NOT descendant of grandparent
	if isDescendantOf(grandparent, unrelated) {
		t.Errorf("Expected unrelated NOT to be descendant of grandparent")
	}

	// Test: nil checks
	if isDescendantOf(nil, child) {
		t.Errorf("Expected false when ancestor is nil")
	}
	if isDescendantOf(grandparent, nil) {
		t.Errorf("Expected false when potentialDescendant is nil")
	}
}

func TestRemoveFromParent(t *testing.T) {
	worklog := createTestWorklog()

	// Create: parent -> child, and another root task
	parent := NewTask("parent")
	child := NewTask("child")
	rootTask := NewTask("rootTask")

	worklog.tasks = append(worklog.tasks, parent, rootTask)
	parent.children = append(parent.children, child)
	child.parent = parent

	// Remove child from parent
	worklog.removeFromParent(child)

	// Verify child has no parent
	if child.parent != nil {
		t.Errorf("Expected child to have no parent after removal")
	}

	// Verify parent no longer has child
	if len(parent.children) != 0 {
		t.Errorf("Expected parent to have no children")
	}

	// Verify root task is still in worklog.tasks
	if len(worklog.tasks) != 2 {
		t.Errorf("Expected worklog to still have 2 root tasks (parent + rootTask)")
	}

	// Now remove rootTask (from root)
	worklog.removeFromParent(rootTask)

	// Verify rootTask is removed from worklog.tasks
	if len(worklog.tasks) != 1 {
		t.Errorf("Expected worklog to have 1 root task after removing rootTask")
	}
	if worklog.tasks[0].uuid != parent.uuid {
		t.Errorf("Expected remaining root task to be parent")
	}
}

func TestUpdateTaskParentRemainsUnchangedOnError(t *testing.T) {
	worklog := createTestWorklog()

	// Create: parent -> child
	parent := NewTask("parent")
	child := NewTask("child")

	worklog.tasks = append(worklog.tasks, parent)
	parent.children = append(parent.children, child)
	child.parent = parent

	// Attempt to move parent to child (cycle) - should fail
	err := worklog.UpdateTask(parent.uuid, TaskUpdate{
		ParentUUID: strPtr(child.uuid),
	})
	if err == nil {
		t.Errorf("Expected error")
	}

	// Verify parent's structure is unchanged
	if parent.parent != nil {
		t.Errorf("Expected parent to still be at root")
	}
	if len(parent.children) != 1 || parent.children[0].uuid != child.uuid {
		t.Errorf("Expected parent to still have child as its only child")
	}
	if child.parent != parent {
		t.Errorf("Expected child's parent to still be parent")
	}
}

func TestDeleteTask(t *testing.T) {
	worklog := createTestWorklog()

	parent := NewTask("parent")
	child := NewTask("child")
	grandchild := NewTask("grandchild")
	sibling := NewTask("sibling")

	worklog.tasks = append(worklog.tasks, parent, sibling)
	parent.children = append(parent.children, child)
	child.parent = parent
	child.children = append(child.children, grandchild)
	grandchild.parent = child

	deleted, err := worklog.DeleteTask(parent.uuid)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if deleted != 3 {
		t.Errorf("Expected 3 deleted tasks, got %d", deleted)
	}

	// parent should be gone from root
	if len(worklog.tasks) != 1 || worklog.tasks[0].uuid != sibling.uuid {
		t.Errorf("Expected only sibling at root")
	}

	// sibling should be unaffected
	if sibling.parent != nil {
		t.Errorf("Expected sibling to have no parent")
	}

	// parent and its descendants should not be findable
	if worklog.FindTaskByUUID(parent.uuid) != nil {
		t.Errorf("Expected parent to be gone")
	}
	if worklog.FindTaskByUUID(child.uuid) != nil {
		t.Errorf("Expected child to be gone")
	}
	if worklog.FindTaskByUUID(grandchild.uuid) != nil {
		t.Errorf("Expected grandchild to be gone")
	}
}

func TestDeleteTaskLeaf(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("leaf")
	worklog.tasks = append(worklog.tasks, task)

	deleted, err := worklog.DeleteTask(task.uuid)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted task, got %d", deleted)
	}

	if len(worklog.tasks) != 0 {
		t.Errorf("Expected no root tasks")
	}
}

func TestDeleteTaskNonExistent(t *testing.T) {
	worklog := createTestWorklog()

	deleted, err := worklog.DeleteTask("non-existent-uuid")
	if err == nil {
		t.Errorf("Expected error for non-existent task")
	}

	if deleted != 0 {
		t.Errorf("Expected 0 deleted tasks, got %d", deleted)
	}
}
