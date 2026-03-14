<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that an error is raised if the output method is HTML and &gt; appears within a processing instruction in the result tree. -->

   <t:output method="html" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <doc>hello
	<t:processing-instruction name="PI">&gt;</t:processing-instruction>
      </doc>
   </t:template>
</t:transform>
