<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if  @normalization-form="fully-normalized" and the result begins with a combining character.-->

   <t:output method="xml"
             encoding="UTF-8"
             indent="no"
             normalization-form="fully-normalized"/>

   <t:template match="/">
	     <doc>́hello</doc>
   </t:template>
</t:transform>
