<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration with a top-level
         element in an alien prefixed namespace -->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
            <z:html xmlns:z="http://www.example.com/not-xhtml">
                <z:head>
                    <z:title>Not XHTML!</z:title>
                </z:head>
                <z:body>
                    <z:p>But what is it then?</z:p>
                </z:body>
            </z:html>
    </t:template>
</t:transform>
