<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if the value of 
  				@encoding in output method HTML is not supported.-->

   <t:output method="html" encoding="XXX-xx" indent="no"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
