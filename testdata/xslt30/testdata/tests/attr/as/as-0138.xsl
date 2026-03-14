<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://myfunc.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test with xsl:function that has @as="attribute()?", 
  				and returns nothing or an xsl:attribute. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:function name="my:func1" as="attribute()?"> </t:function>

   <t:function name="my:func2" as="attribute()?">
      <t:copy-of select="document('as-15.xml')/doc/item/@attrib"/>
   </t:function>

   <t:template match="/doc">
      <out>
         <func1>
            <t:value-of select="my:func1() instance of attribute()?"/>
            <t:value-of select="my:func1() instance of attribute()"/>
         </func1>
         <func2>
            <t:value-of select="my:func2() instance of attribute()?"/>
            <t:value-of select="my:func2() instance of attribute()"/>
         </func2>
      </out>
   </t:template>
</t:transform>
