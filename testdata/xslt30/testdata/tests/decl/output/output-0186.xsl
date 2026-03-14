<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if @omit-xml-declarations="yes" in xsl:output 
  				and @standalone is other than 'omit'.-->

   <t:output method="xml"
             encoding="UTF-8"
             indent="no"
             omit-xml-declaration="yes"
             standalone="yes"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
