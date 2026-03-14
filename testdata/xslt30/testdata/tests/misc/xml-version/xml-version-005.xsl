<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that LSEP (0x2028) can appear in a text node.  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
		    <out>
         <t:text>Test character: </t:text>
         <t:text>&#x2028;</t:text>
      </out>
  </t:template>
</t:transform>
