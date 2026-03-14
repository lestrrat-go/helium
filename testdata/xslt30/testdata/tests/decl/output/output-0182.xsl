<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if @standalone value is other than 
  				omit and the instance of the data model contains multiple elements children of the root node.-->

   <t:output method="xml" encoding="UTF-8" indent="no" standalone="yes"/>

   <t:template match="/">
	     <doc>hello</doc>
	     <doc>hello</doc>
	     <doc>hello</doc>
   </t:template>
</t:transform>
