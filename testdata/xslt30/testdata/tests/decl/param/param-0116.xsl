<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that a dynamic error is raised if the initial 
  				template defines a template parameter that specifies required="yes". -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
	     <out/>
   </t:template>

   <t:template name="temp">
	     <t:param name="a" required="yes"/>	
   </t:template>
</t:transform>
