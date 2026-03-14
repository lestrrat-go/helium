<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses some of the String functions to create a text node. Functions used:
				fn:string-join
				fn:upper-case
				fn:lower-case -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:param name="par" select="'foo:'"/>

   <t:variable name="var" select="'abc'"/>

   <t:variable name="sep" select="'**'"/>

   <t:template match="doc">
  		  <out>
         <t:text>
</t:text>
         <t:value-of select="string-join(('What', 'a', 'great', 'test', 'case', '...') ,$sep)"/>
         <t:text>
</t:text>
         <t:value-of select="upper-case(substring('Once upon a time',13))"/>
         <t:text>
</t:text>
         <t:value-of select="lower-case(substring('once upon a time',1,4))"/>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
