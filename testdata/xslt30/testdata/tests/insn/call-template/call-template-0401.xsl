<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Launched with initial template and that template has a param marked required.
  			   (Processors not permitted to supply template params at launch, only stylesheet 
  			   params.) -->

   <t:template match="/">
	 	   <t:comment>ERROR: this value should NOT appear</t:comment>
	  </t:template>

   <t:template name="temp">
		    <t:param name="needed" required="yes"/>
	  	  <out>
         <t:value-of select="."/>
      </out>
	  </t:template>
</t:transform>
