
package main


// ================ Habit Tracker CLI =================

// Project Goal:
// 	Build a command-line tool to help users track daily habits, using Go OOP concepts (structs, methods, interfaces, composition).

// Instructions (Apply Week-03-OOPS Concepts!):
// 1. Define a Habit struct with fields for name, streak, and last completed date.
// 2. Create methods for Habit (e.g., MarkComplete, ResetStreak, Display).
// 3. Use a slice to store multiple habits.
// 4. Define a HabitManager struct to manage habits (add, remove, list, update).
// 5. Implement interfaces for displaying and updating habits.
// 6. Use struct composition if needed (e.g., embedding for advanced features).
// 7. Parse user commands (add, complete, list, reset) from command-line arguments or interactive input.
// 8. Print output to the console in a user-friendly format.

// Input Example:
// 	$ go run main.go add "Drink Water"
// 	$ go run main.go complete "Drink Water"
// 	$ go run main.go list

// Output Example:
// 	Habit: Drink Water | Streak: 5 | Last Completed: 2026-01-28

// Tips:
// - Use structs for modeling habits and managers.
// - Use methods to encapsulate behavior.
// - Use interfaces for extensibility (e.g., Displayable, Updatable).
// - Use composition for advanced features (e.g., notifications).
// - Add error handling for invalid commands or missing habits.

// Success Criteria:
// - Uses structs, methods, interfaces, and composition.
// - Runs from the command line with arguments.
// - Can add, complete, list, and reset habits.
// - Handles errors gracefully.


// Write your code below this line

// TODO: Define Habit struct
type Habit struct {

}


// TODO: Implement methods for Habit (MarkComplete, ResetStreak, Display)
func (h *Habit) MarkComplete() bool {
	return false
}

func (h *Habit) ResetStreak() bool {
	return false
}

func (h *Habit) Display() string {
	return ""
}

// TODO: Define HabitManager struct to manage habits
type HabitManager struct {

}

// TODO: Implement methods for HabitManager (AddHabit, CompleteHabit, ListHabits, ResetHabit)
func (hm *HabitManager) AddHabit() {

}

func (hm *HabitManager) CompleteHabit() {
	
}

func (hm *HabitManager) ListHabits() {
	
}

func (hm *HabitManager) ResetHabit() {
	
}

// TODO: Define interfaces for displaying and updating habits
type Displayable interface {
	
}

// TODO: Use composition for advanced features (optional)

// TODO: Parse user commands and arguments

// TODO: Print output to the console


func main() {

}