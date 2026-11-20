# Code Conversion System - iOS/Kotlin to Flutter

## Overview

The Agentic Software Factory now includes a **Code Conversion System** that automatically converts iOS (Swift) and Kotlin (Android) code to equivalent Flutter (Dart) code. This system combines an intuitive web UI with intelligent agent-driven backend processing.

## Architecture

### Components

```
User Interface (React/TypeScript)
    ↓
API Client (FactoryClient)
    ↓
Backend HTTP API (Go)
    ↓
Conversion Handlers
    ↓
Agent Processing
```

### Technology Stack

**Frontend:**
- React 18 with TypeScript
- Tailwind CSS for styling
- Side-by-side code comparison with syntax highlighting
- Real-time status updates (polls every 5 seconds)

**Backend:**
- Go HTTP handlers for conversion operations
- In-memory conversion store (can be extended to database)
- Support for concurrent conversions

## Features

### 1. Single Code Snippet Conversion
Users can submit individual code snippets:

```swift
// User submits Swift code
import UIKit

class ViewController: UIViewController {
    @IBOutlet weak var label: UILabel!
    
    override func viewDidLoad() {
        super.viewDidLoad()
        label.text = "Hello, World!"
    }
}
```

**Output:** Equivalent Dart/Flutter code
```dart
import 'package:flutter/material.dart';

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        body: Center(
          child: Text('Hello, World!'),
        ),
      ),
    );
  }
}
```

### 2. Multi-File Project Support
Support for uploading and converting entire project structures:

- Multiple source files processed together
- Dependency resolution
- Architecture-aware conversion
- Full project structure preservation

### 3. Supported Conversions

| Source | Target | Status |
|--------|--------|--------|
| Swift (iOS) | Dart (Flutter) | ✅ Supported |
| Kotlin (Android) | Dart (Flutter) | ✅ Supported |

## API Endpoints

### Submit Code for Conversion

```http
POST /api/conversions

Content-Type: application/json

{
  "sourceLanguage": "swift",
  "sourceCode": "import UIKit\n\nclass ViewController: UIViewController { ... }"
}
```

**Response (201 Created):**
```json
{
  "id": "20261119154530-abc12345",
  "sourceLanguage": "swift",
  "sourceCode": "import UIKit\n...",
  "status": "pending",
  "createdAt": "2026-11-19T15:45:30Z"
}
```

### Get Conversion Status

```http
GET /api/conversions/{id}
```

**Response:**
```json
{
  "id": "20261119154530-abc12345",
  "sourceLanguage": "swift",
  "sourceCode": "...",
  "flutterCode": "import 'package:flutter/material.dart';\n...",
  "status": "completed",
  "assignedAgent": "converter-1",
  "createdAt": "2026-11-19T15:45:30Z",
  "completedAt": "2026-11-19T15:46:15Z"
}
```

### Update Conversion Status

```http
PATCH /api/conversions/{id}

Content-Type: application/json

{
  "status": "completed",
  "flutterCode": "...",
  "assignedAgent": "converter-1"
}
```

### List All Conversions

```http
GET /api/conversions
```

**Response:**
```json
{
  "conversions": [
    { "id": "...", "sourceLanguage": "swift", "status": "completed", ... },
    { "id": "...", "sourceLanguage": "kotlin", "status": "converting", ... }
  ],
  "count": 2
}
```

## Conversion Status Workflow

### Status States

```
pending → converting → completed
                    ↘ failed
```

### State Descriptions

- **pending**: Conversion submitted, waiting for agent assignment
- **converting**: Agent is actively processing the code
- **completed**: Conversion finished successfully, flutterCode populated
- **failed**: Conversion failed, error field contains details

## UI Usage

### Submitting Code

1. Navigate to **Code Converter** in sidebar (Zap icon)
2. Select source language:
   - **iOS (Swift)** for Objective-C/Swift code
   - **Kotlin (Android)** for Kotlin/Android code
3. Paste source code into textarea
4. Click **Convert to Flutter** button
5. Status updates appear in real-time

### Viewing Results

Each completed conversion displays:

- **Source Code Block** (left): Original iOS/Kotlin code with syntax highlighting
- **Flutter Code Block** (right): Converted Dart/Flutter code with syntax highlighting
- **Action Buttons**:
  - **Copy to Clipboard**: Copy the converted code
  - **Download Flutter Code**: Download as `converted.dart` file

### Conversion History

- Real-time polling updates status every 5 seconds
- Shows all submitted conversions (pending, converting, completed, failed)
- Filter by status or language
- Timestamps and agent information displayed

## Tips for Best Results

1. **Include Complete Blocks**: Submit entire classes, functions, or modules, not snippets
2. **Add Comments**: Comments help the conversion engine understand intent
3. **Manage Complexity**: For large projects, convert by feature or module
4. **Review Thoroughly**: Always review converted code for Flutter best practices
5. **Test Carefully**: Run the converted code in a Flutter environment before production

## Example Conversions

### iOS UIViewController → Flutter StatelessWidget

**iOS:**
```swift
import UIKit

class ListViewController: UIViewController {
    @IBOutlet weak var tableView: UITableView!
    var items: [String] = []
    
    override func viewDidLoad() {
        super.viewDidLoad()
        tableView.dataSource = self
        loadItems()
    }
    
    func loadItems() {
        items = ["Item 1", "Item 2", "Item 3"]
        tableView.reloadData()
    }
}

extension ListViewController: UITableViewDataSource {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return items.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "Cell", for: indexPath)
        cell.textLabel?.text = items[indexPath.row]
        return cell
    }
}
```

**Flutter:**
```dart
import 'package:flutter/material.dart';

class ListPage extends StatefulWidget {
  @override
  State<ListPage> createState() => _ListPageState();
}

class _ListPageState extends State<ListPage> {
  List<String> items = [];

  @override
  void initState() {
    super.initState();
    loadItems();
  }

  void loadItems() {
    setState(() {
      items = ["Item 1", "Item 2", "Item 3"];
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text("Items")),
      body: ListView.builder(
        itemCount: items.length,
        itemBuilder: (context, index) {
          return ListTile(
            title: Text(items[index]),
          );
        },
      ),
    );
  }
}
```

### Kotlin Activity → Flutter StatefulWidget

**Kotlin:**
```kotlin
import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import android.widget.EditText
import android.widget.Button
import android.widget.Toast

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        
        val inputField = findViewById<EditText>(R.id.input)
        val submitBtn = findViewById<Button>(R.id.submit)
        
        submitBtn.setOnClickListener {
            val text = inputField.text.toString()
            if (text.isNotEmpty()) {
                Toast.makeText(this, "Submitted: $text", Toast.LENGTH_SHORT).show()
            }
        }
    }
}
```

**Flutter:**
```dart
import 'package:flutter/material.dart';

class HomePage extends StatefulWidget {
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final TextEditingController _controller = TextEditingController();

  void _submit() {
    final text = _controller.text.trim();
    if (text.isNotEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("Submitted: $text")),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text("Home")),
      body: Padding(
        padding: EdgeInsets.all(16),
        child: Column(
          children: [
            TextField(
              controller: _controller,
              decoration: InputDecoration(hintText: "Enter text"),
            ),
            SizedBox(height: 16),
            ElevatedButton(
              onPressed: _submit,
              child: Text("Submit"),
            ),
          ],
        ),
      ),
    );
  }
}
```

## Agent Integration

### How Agents Process Conversions

1. **Polling**: Agents continuously poll `/api/conversions` for pending tasks
2. **Assignment**: Agent claims a conversion by updating status to "converting"
3. **Processing**: Agent converts code using its conversion engine
4. **Completion**: Agent updates conversion with flutterCode and status="completed"

### Example Agent Workflow

```go
// Agent picks up pending conversion
func agentConversionLoop() {
    conversions, _ := client.ListConversions()
    for _, conversion := range conversions {
        if conversion.Status == "pending" {
            // Claim the conversion
            client.UpdateConversion(conversion.ID, ConversionUpdate{
                Status: "converting",
                AssignedAgent: "converter-1",
            })
            
            // Convert code
            flutterCode := convertToFlutter(conversion.SourceLanguage, conversion.SourceCode)
            
            // Update with results
            client.UpdateConversion(conversion.ID, ConversionUpdate{
                Status: "completed",
                FlutterCode: flutterCode,
            })
        }
    }
}
```

## Performance Considerations

- **Conversion Time**: Typically 1-5 seconds per conversion
- **Code Size Limits**: Recommend <5000 lines per submission
- **Concurrency**: Backend supports unlimited concurrent conversions
- **Storage**: Conversions stored in memory (extend with database for persistence)

## Future Enhancements

- [ ] Database persistence for conversions
- [ ] Support for ReactNative code
- [ ] Support for Web (HTML/CSS/JS) to Flutter
- [ ] Batch processing with job queues
- [ ] Advanced syntax highlighting
- [ ] Conversion templates and snippets
- [ ] AI-powered complexity scoring
- [ ] Detailed conversion analysis and warnings
- [ ] Support for framework-specific conversions (Jetpack Compose, SwiftUI)

## Troubleshooting

### Conversion Stuck in "Converting" State
- Check if agent process is running
- Verify backend connectivity
- Check server logs for errors

### Invalid Source Code Error
- Ensure code is syntactically valid
- Use complete code blocks (include imports)
- Verify language selection matches code

### Missing Flutter Code in Response
- Conversion may still be processing (check UI for status)
- Code may be too complex (try smaller snippets)
- Check error field for details

## Configuration

### Backend Configuration

```go
// In main.go, wire up conversion handlers
router.HandleFunc("POST /conversions", handlers.SubmitCodeConversion)
router.HandleFunc("GET /conversions", handlers.ListConversions)
router.HandleFunc("GET /conversions/{id}", handlers.GetConversion)
router.HandleFunc("PATCH /conversions/{id}", handlers.UpdateConversion)
```

### Frontend Configuration

```typescript
// In web/src/api/client.ts
const client = new FactoryClient('http://localhost:8080/api');
```

## Related Documentation

- [Agentic Software Factory README](../README.md)
- [Phase 5: Web Control Panel](./PHASE5_WEB_UI.md)
- [Issues and Projects Management](../README.md#github-style-features)
