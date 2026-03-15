<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if in xsl:output @method="html" and the value of @version is not supported by the serializer.-->

   <t:output method="html" encoding="UTF-8" indent="no" version="0.0"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
