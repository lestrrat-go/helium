<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if the value of @normalization-form in output method XHTML is not supported by the serializer.-->

   <t:output method="xhtml"
             encoding="UTF-8"
             indent="no"
             normalization-form="ABC"/>

   <t:template match="/">
	     <doc>hello</doc>
   </t:template>
</t:transform>
