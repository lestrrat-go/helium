<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if the output method is HTML and the result tree contains characters in the #x7F-#x9F range.-->

   <t:output method="html" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <doc>&#x9f;</doc>
   </t:template>
</t:transform>
