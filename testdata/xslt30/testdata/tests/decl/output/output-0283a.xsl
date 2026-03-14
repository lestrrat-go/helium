<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test that an error is raised on invalid @undeclare-prefixes value ("True") in xsl:output.-->

   <t:output method="xml"
             version="1.1"
             encoding="UTF-8"
             indent="no"
             undeclare-prefixes="True"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
