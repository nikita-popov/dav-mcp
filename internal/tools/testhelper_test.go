package tools_test

// Shared CalDAV discovery XML bodies used by connectCalDAV, connectJournal,
// and connectTodo. <c:calendar/> inside <resourcetype> is required so that
// dav.DiscoverCollections recognises the collection as a calendar.

const testPrincipalBody = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response><href>/</href>
    <propstat><prop><current-user-principal><href>/principals/user/</href></current-user-principal></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

const testCalHomeBody = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response><href>/principals/user/</href>
    <propstat><prop><c:calendar-home-set><href>/calendars/user/</href></c:calendar-home-set></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

// testCollectionsBody returns a PROPFIND depth:1 response for /calendars/user/
// with a single collection that advertises the given component (e.g. "VEVENT",
// "VTODO", "VJOURNAL"). Passing an empty string omits the component set,
// which is sufficient for plain calendar collections (VEVENT).
func testCollectionsBody(component string) string {
	compSet := ""
	if component != "" {
		compSet = `\n      <c:supported-calendar-component-set><c:comp name="` + component + `"/></c:supported-calendar-component-set>`
	}
	return `<?xml version="1.0"?>` + "\n" +
		`<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">` + "\n" +
		`  <response><href>/calendars/user/personal/</href>` + "\n" +
		`    <propstat><prop>` + "\n" +
		`      <displayname>Personal</displayname>` + "\n" +
		`      <resourcetype><collection/><c:calendar/></resourcetype>` + compSet + "\n" +
		`    </prop>` + "\n" +
		`    <status>HTTP/1.1 200 OK</status></propstat>` + "\n" +
		`  </response>` + "\n" +
		`</multistatus>`
}
