<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses use-when on an element while the element has a static error 
  				and the EBV evaluates to true and therefore the element should not be excluded 
  				from the module. This is to verify that the error is reported.  -->

   <t:template match="doc">
  	   <out>
         <t:text>
</t:text>
         <t:element name="elem" select="'invalid select attribute'" use-when="true()"/>
         <t:text>
</t:text>
      </out>
  </t:template>
</t:transform>
