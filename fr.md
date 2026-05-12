**3.2.1** **Functional** **Requirements**


Table 3: Enterprise Management and Multi-Tenancy Requirements














|Req. ID|Requirement Title|Description|Priority|VM|
|---|---|---|---|---|
|FR-MT-001|Multi-tenant<br>architecture|The system shall completely isolate data, confgurations,<br>users, exams, and results between diferent<br>enterprises/organizations.|Mandatory|Test / Inspection|
|FR-MT-002|Enterprise registration|The system shall allow organizations to self-register with<br>company details, contact person, and desired subscription<br>plan.|Mandatory|Test|
|FR-MT-003|Enterprise account<br>approval|System administrator shall review and approve/reject<br>enterprise registration requests.|Mandatory|Demonstration|
|FR-MT-004|Enterprise profle<br>management|Approved enterprises shall be able to update their profle,<br>branding (logo, colors), and contact details.|Mandatory|Test|
|FR-MT-005|Subscription tier<br>enforcement|The system shall enforce feature restrictions based on the<br>enterprise’s subscription tier (Basic, Premium, Enterprise).|Mandatory|Test|
|FR-MT-006|Tenant<br>Deletion/Suspension|System administrator shall be able to suspend an<br>enterprise account (locking out all associated users) and/or<br>permanently delete an enterprise and all associated data<br>after a 90-day retention period|Mandatory|Demonstration|


Table 4: Authentication and User Management Requirements








|Req. ID|Requirement Title|Description|Priority|VM|
|---|---|---|---|---|
|FR-AUTH-001|Role-based access<br>control|The system shall support roles: Super Admin, Enterprise Admin,<br>Enterprise Staf, Candidate (temporary).|Mandatory|Test|
|FR-AUTH-002|Enterprise-issued<br>credential login|Candidates shall log in using enterprise-issued unique token or ID<br>(no personal email/password required).|Mandatory|Test|
|FR-AUTH-003|Enterprise admin/user<br>management|Enterprise admins shall create/manage their staf accounts and<br>permissions.|Mandatory|Test|
|FR-AUTH-004|Face registration during<br>exam onboarding|During exam start, candidates shall submit a live facial image that<br>is stored as reference for the session.|Mandatory|Test|
|FR-AUTH-005|Periodic face verifcation<br>(Premium+)|The system shall periodically capture facial images during exam and<br>compare with registration image using face recognition (available<br>only for Premium/Enterprise plans).|Desirable|Test|
|FR-AUTH-006|Session timeout &<br>re-authentication|Inactive sessions shall timeout after a confgurable period and<br>require re-authentication.|Mandatory|Test|



Table 5: Examination Creation & Management Requirements


**Req.** **ID** **Requirement** **Title** **Description** **Priority** **VM**


|FR-EXM-001|Question Bank &<br>Metadata Management|Enterprise Admins shall create, categorize, and manage<br>central question banks. The system must support the<br>following question types: Multiple Choice (MCQ),<br>True/False, Short Answer, and Essay. Each question must<br>be assigned mandatory metadata for Topic/Category,<br>Difficulty Level (e.g., Easy, Medium, Hard), and Media<br>Support (allowing image, audio, or video attachments).|Mandatory|Test|
|---|---|---|---|---|
|FR-EXM-002|Advanced Exam<br>Confguration Wizard|The system shall provide a multi-step wizard to create<br>exams with comprehensive settings: overall Exam<br>Duration, Passing Score percentage, Negative Marking<br>rules, and Targeted Randomization. Targeted<br>Randomization must allow the Admin to defne the exam<br>structure by drawing N questions from specifc categories or<br>difculty levels (e.g., 5 Easy questions from ”Topic A,” 10<br>Medium questions from ”Topic B”).|Mandatory|Demonstration|
|FR-EXM-003|Strict Scheduling &<br>Status Control|Admins shall precisely schedule exams with defnite start<br>date/time and end date/time, defne available Invitation<br>Methods (Email, Link, Token), and set the Maximum<br>Allowed Participants. The system must automatically<br>manage and display the exam’s status lifecycle: Scheduled,<br>Active, Closed, and Archived.|Mandatory|Test|
|FR-EXM-004|Bulk Candidate<br>Enrollment & Credential<br>Generation|Admins shall upload a CSV fle or manually input<br>candidate details. The system shall validate the uploaded<br>data (e.g., checking for unique identifers) and<br>automatically generate unique, secure exam links or tokens<br>for each candidate, sending the credentials via email.|Mandatory|Test|


FR-EXM-005 Exam Template Cloning
& Reuse



Admins shall be able to save any configured exam as a
reusable template for future use. The system must also
provide a Cloning function to duplicate all settings,
question selections, and organization branding from any
existing exam (Active or Closed) into a new, editable draft.



Mandatory Demonstration



Table 6: Candidate Examination Interface Requirements








|Req. ID|Requirement Title|Description|Priority|VM|
|---|---|---|---|---|
|FR-CAND-001|Secure Web Browser<br>Mode & Warning|The exam interface shall enforce full-screen mode and use<br>JavaScript/browser APIs to disable right-click,<br>copy-paste, and common browser shortcuts (e.g.,<br>Ctrl+C). A prominent, persistent warning banner shall<br>inform the candidate that switching tabs/applications will<br>be logged and may lead to disqualifcation.|Mandatory|Test|
|FR-CAND-002|Server-Enforced<br>Real-Time Timer|A strict countdown timer shall be displayed prominently.<br>The remaining time shall be tracked and strictly enforced<br>server-side to prevent client-side manipulation. The server<br>shall reject answers submitted even milliseconds after the<br>scheduled end time.|Mandatory|Test|
|FR-CAND-003|Question Navigation &<br>Review|Candidates shall navigate questions sequentially or via a<br>navigation index. They must be able to mark questions<br>for review and receive a clear confrmation prompt before<br>fnal submission or when time expires.|Mandatory|Demonstration|


|FR-CAND-004|Auto-Save & Submission<br>Confri mation|Answers shall be auto-saved locally (using<br>IndexedDB/LocalStorage) and synced to the server at<br>least every 60 seconds. A comprehensive submission<br>confri mation screen detailing the number of<br>answered/unanswered questions must appear before final<br>submission.|Mandatory|Test|
|---|---|---|---|---|
|FR-CAND-005|Graceful Handling of<br>Connectivity Issues|The system shall detect loss of connection and<br>immediately queue answers locally using an ofine-frst<br>approach (Service Workers/IndexedDB). When the<br>connection restores, the system shall automatically and<br>silently sync all queued data to the server and notify the<br>candidate.|Mandatory|Test|
|FR-CAND-006|Accessibility &<br>Readability Controls|The interface shall conform to at least WCAG 2.1 AA<br>standards. Candidates shall have simple controls to adjust<br>font size, color contrast (dark/light mode), and use<br>standard keyboard navigation for accessibility.|Mandatory|Inspection|
|FR-CAND-007|Secure Session<br>Termination|If a major proctoring violation (e.g., identity mismatch<br>(FR-PROC-004)) is fagged, the system shall provide the<br>Enterprise Admin the option to force-terminate the<br>candidate’s session immediately. The candidate must<br>receive a non-specifc notifcation of termination.|Mandatory|Demonstration|



Table 7: AI Proctoring & Monitoring Requirements


|Req. ID|Requirement Title|Description|Priority|VM|
|---|---|---|---|---|
|FR-PROC-001|Tab/window switch<br>detection|The system shall detect and log every tab/window switch or<br>application change during exam.|Mandatory|Test|
|FR-PROC-002|Mouse activity<br>monitoring|The system shall monitor mouse movement patterns and fag<br>prolonged inactivity or unusual behavior.|Mandatory|Test|
|FR-PROC-003|Webcam-based face<br>detection|System shall continuously verify that a face is present in the<br>webcam feed (Basic+ plans).|Mandatory|Test|
|FR-PROC-004|Facial identity<br>verifcation (Premium+)|The system shall periodically verify that the detected face<br>matches the registered face (Premium/Enterprise only).|Desirable|Test|
|FR-PROC-005|Multiple face detection|The system shall fag if more than one face is detected in the<br>frame.|Desirable|Test|
|FR-PROC-006|Proctoring event logging|All proctoring events shall be logged with timestamp, severity,<br>and optional screenshot.|Mandatory|Inspection|
|FR-PROC-007|Cheating probability<br>score|The system shall calculate an overall cheating probability score<br>based on all proctoring events.|Mandatory|Test|


Table 8: Grading, Analytics & Certificate Requirements





**Req.** **ID** **Requirement** **Title** **Description** **Priority** **VM**


|FR-GRAD-001|Automatic grading for<br>subjective/objective<br>questions|The system shall automatically grade MCQ, True/False,<br>fli l-in-the-blank, short answer, and essay questions using<br>confgi urable rubrics and AI assistance where applicable.|Mandatory|Test|
|---|---|---|---|---|
|FR-GRAD-002|Blind grading|When a subjective answer is routed for mandatory human<br>review (due to low AI confdence), the reviewer shall only<br>see the candidate’s response and the question rubric and<br>must not see the candidate’s name or any AI-assigned<br>score until the manual grade is submitted.|Mandatory|Demonstration|
|FR-GRAD-003|AI-assisted result<br>analytics|The system shall provide detailed analytics including<br>score distribution, time per question, cheating score<br>impact, and rankings.|Mandatory|Demonstration|
|FR-GRAD-004|Personalized feedback|Candidates shall receive personalized feedback reports<br>including score, correct answers, detected cheating fags,<br>and relative ranking.|Desirable|Demonstration|
|FR-GRAD-005|Automatic certifcate<br>generation|The system shall generate and deliver customizable PDF<br>certifcates to passing candidates with organization<br>branding and a QR code for verifcation.|Mandatory|Test|


Table 9: Dashboard & Reporting Requirements



**Req.** **ID** **Requirement** **Title** **Description** **Priority** **VM**


|FR-DASH-001|Role-Based Dashboards<br>& Customization|Super Admin, Enterprise Admin, and Staff shall<br>have customized, widget-based dashboards showing<br>relevant key metrics (KPIs) and quick links.<br>Enterprise Admins shall be able to confgi ure or<br>reorder the widgets displayed on their dashboard.|Mandatory|Demonstration|
|---|---|---|---|---|
|FR-DASH-002|Real-Time Exam<br>Monitoring Dashboard|Enterprise Admins shall view live exams, active<br>candidates, and proctoring fags in real-time. The<br>dashboard shall feature low-latency updates (max<br>5-second latency) and color-coded severity<br>indicators for proctoring events.|Mandatory|Demonstration / Test|
|FR-DASH-003|Asynchronous<br>Comprehensive<br>Reporting|The system shall provide comprehensive reports for<br>exam results, proctoring logs, and candidate<br>performance. For reports exceeding 500 records,<br>generation shall be asynchronous (user is notifed<br>via email/in-app when the report is ready).|Mandatory|Test|
|FR-DASH-004|Granular Data Export<br>Formats|Reports shall be exportable in multiple formats:<br>PDF (formatted, summary), Excel/CSV (raw data<br>for analysis), and JSON/API endpoint (for<br>integration with Enterprise BI tools -<br>Enterprise-tier feature).|Mandatory|Test|
|FR-DASH-005|Audit Log & Activity<br>Tracking|Super Admins and Enterprise Admins shall have a<br>dedicated report/dashboard to view a complete,<br>immutable audit trail of all critical actions (e.g.,<br>user creation, exam changes, deletion, payment<br>history) with timestamp and actor details.|Mandatory|Inspection|


|FR-DASH-006|Performance & Trend<br>Analytics|Enterprise Admins shall access analytics<br>dashboards to track historical trends (e.g.,<br>month-over-month exam volume, average scores<br>over time) and view performance benchmarks<br>against other Enterprise exams (anonymously).|Desirable|Demonstration|
|---|---|---|---|---|
|FR-DASH-007|Custom Report Builder|The system shall provide an interface that allows<br>Enterprise Admins to defne and save custom<br>reports by selecting specifc flters, felds, and<br>grouping criteria (e.g., performance by<br>department/location).|Desirable|Demonstration|


Table 10: Payment & Subscription Requirements












|Req. ID|Requirement Title|Description|Priority|VM|
|---|---|---|---|---|
|FR-PAY-001|Gateway Integration &<br>Localization|The system shall integrate with the specifed<br>Ethiopian payment gateways (Telebirr, CBE Birr)<br>and support local currency (ETB) transactions.|Mandatory|Test|
|FR-PAY-002|Secure Transaction<br>Handling|The system shall comply with PCI DSS Level 2 (or<br>equivalent standard for handling payment<br>tokens/data) and ensure all payment data<br>transmission is encrypted using TLS 1.2 or higher.|Mandatory|Inspection / Audit|


|FR-PAY-003|Subscription Plan<br>Management|Super Admin shall defni e, modify, activate, and<br>deactivate subscription plans, including features,<br>usage limits (max users, max exams), and pricing<br>structures.|Mandatory|Demonstration|
|---|---|---|---|---|
|FR-PAY-004|Automatic Recurring<br>Billing & Dunning|The system shall automatically handle recurring<br>billing and renewal. It must implement a dunning<br>process to manage failed payments, including retries<br>and automated notifcation of the Enterprise Admin.|Mandatory|Test|
|FR-PAY-005|Upgrade/Downgrade<br>Management|The system shall allow Enterprise Admins to upgrade<br>or downgrade their subscription plan, with proration<br>calculated and applied automatically for the<br>remaining period of the current billing cycle.|Mandatory|Test|
|FR-PAY-006|Invoice & Tax<br>Compliance|The system shall automatically generate and deliver<br>a detailed, legally compliant invoice (including<br>applicable Ethiopian taxes or VAT) upon every<br>payment or billing event (new subscription, renewal,<br>proration).|Mandatory|Test / Demonstration|
|FR-PAY-007|Billing History &<br>Management|Enterprise Admins shall have a dedicated portal to<br>view their complete billing history, download all past<br>invoices (FR-PAY-006), and update their payment<br>methods.|Mandatory|Demonstration|
|FR-PAY-008|Account Suspension<br>Workfow|In the event of persistent payment failure (after the<br>dunning process), the system shall automatically<br>trigger the suspension of the Enterprise account,<br>notifying the Enterprise Admin 48 hours prior to the<br>action.|Mandatory|Demonstration|


