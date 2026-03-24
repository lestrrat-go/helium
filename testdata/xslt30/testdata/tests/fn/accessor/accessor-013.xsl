<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of nilled() accessor with node stored in variable -->

   <xslt:template match="doc">
      <out>
         <t1>
            <xslt:variable name="var1" select="processing-instruction()"/>
            <xslt:value-of select="nilled($var1)"/>
         </t1>
         <t2>
            <xslt:variable name="var2" select="comment()"/>
            <xslt:value-of select="nilled($var2)"/>
         </t2>
         <t3>
            <xslt:variable name="var3" select="text()[1]"/>
            <xslt:value-of select="nilled($var3)"/>
         </t3>
         <t4>
            <xslt:variable name="var4" select="element()"/>
            <xslt:value-of select="nilled($var4)"/>
         </t4>
         <t5>
            <xslt:variable name="var5" select="attribute()"/>
            <xslt:value-of select="nilled($var5)"/>
         </t5>
         <t6>
            <xslt:variable name="var6" select="namespace::*"/>
            <xslt:for-each select="$var6">
		             <xslt:value-of select="nilled(.)"/>
		          </xslt:for-each>
         </t6>
      </out>
   </xslt:template>
</xslt:transform>
