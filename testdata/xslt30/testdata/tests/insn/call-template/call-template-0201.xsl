<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Launch transformation with initial named template and also with initial 
  				node specified, and the template references the node (as .) -->

   <t:template match="/">
	 	   <t:comment>ERROR: this value should NOT appear</t:comment>
	  </t:template>

   <t:template name="temp">
 		   <out>
         <t:copy-of select="."/>
      </out>
 	</t:template>
</t:transform>
