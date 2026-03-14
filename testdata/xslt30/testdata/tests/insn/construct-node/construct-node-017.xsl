<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that when there are double hyphens in a comment,
  				a space is inserted after the first hyphen. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc">
  		  <out>
         <t:comment select="'--Valid comment--'"/>
      </out>
	  </t:template>
</t:transform>
