<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration.-->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
        <HTML xmlns="http://www.w3.org/1999/xhtml">
            <HEAD>
                <title>test</title>
            </HEAD>
            <BODY>
                <p>content</p>
            </BODY>
        </HTML>
    </t:template>
</t:transform>
