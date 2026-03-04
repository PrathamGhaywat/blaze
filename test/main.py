#USAGE INSTRUCTIONS:
# 1. Save this code in a file named calculator.py
# 2. Open your terminal/command prompt and navigate to the folder
# 3. Run the script with: python calculator.py
# 4. Enter two numbers when prompted. It will print their sum.

def calculate_sum():
    try:
        # Get user input
        num1 = float(input("Enter first number: "))
        num2 = float(input("Enter second number: "))

        # Calculate result
        result = num1 + num2

        print(f"Result: {result}")
    except ValueError:
        print("Invalid input. Please enter numbers.")

if __name__ == "__main__":
    calculate_sum()