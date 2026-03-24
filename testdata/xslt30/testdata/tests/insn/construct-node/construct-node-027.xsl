<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that there is only one text node created for the literal text.  -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="x" as="item()*">abcdef</t:variable>

   <t:variable name="y">
		    <t:sequence select="('abc', 'processing-instruction()', 'def')"/>
	  </t:variable>

   <t:template match="doc">
		    <t:text>
</t:text>
  		  <out>
         <t:text>
</t:text>
         <x>
            <t:value-of select="count($x)"/>
         </x>
         <t:text>
</t:text>
         <y>
            <t:value-of select="count($y)"/>
         </y>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
