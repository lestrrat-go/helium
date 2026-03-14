<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://myfunc.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test with xsl:function that has @as="element()*" and returns:
				-an empty sequence, one xsl:element node, a sequence of xsl:element nodes -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:function name="my:func1" as="element()*">
      <t:sequence select="()"/>
   </t:function>

   <t:function name="my:func2" as="element()*">
      <t:copy-of select="document('as-15.xml')/doc/item"/>
   </t:function>

   <t:function name="my:func3" as="element()*">
      <t:copy-of select="document('as-15.xml')/doc//*"/>
   </t:function>

   <t:template match="/doc">
      <out>
         <func1>
            <t:value-of select="my:func1() instance of element()*"/>
            <t:value-of select="my:func1() instance of element()"/>
         </func1>
         <func2>
            <t:value-of select="my:func2() instance of element()*"/>
            <t:value-of select="my:func2() instance of element()"/>
         </func2>
         <func3>
            <t:value-of select="my:func3() instance of element()*"/>
            <t:value-of select="my:func3() instance of element()"/>
         </func3>
      </out>
   </t:template>
</t:transform>
