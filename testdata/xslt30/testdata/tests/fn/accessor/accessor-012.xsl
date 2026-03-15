<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of node-name()'s return type as xs:QName -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:variable name="var1" select="processing-instruction()"/>
            <xslt:value-of select="node-name($var1) instance of xs:QName?"/>
         </t1>
         <t2>
            <xslt:variable name="var2" select="comment()"/>
            <xslt:value-of select="node-name($var2) instance of xs:QName?"/>
         </t2>
         <t3>
            <xslt:variable name="var3" select="text()[1]"/>
            <xslt:value-of select="node-name($var3) instance of xs:QName?"/>
         </t3>
         <t4>
            <xslt:variable name="var4" select="element()"/>
            <xslt:value-of select="node-name($var4) instance of xs:QName?"/>
         </t4>
         <t5>
            <xslt:variable name="var5" select="attribute()"/>
            <xslt:value-of select="node-name($var5) instance of xs:QName?"/>
         </t5>
      </out>
   </xslt:template>
</xslt:transform>
