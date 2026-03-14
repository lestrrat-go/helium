<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised when in xsl:output @method="xml", @undeclare-prefixes="yes", and @version="1.0"-->

   <t:output method="xml"
             encoding="UTF-8"
             indent="no"
             undeclare-prefixes="yes"
             version="1.0"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
