<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of nilled() accessor with node stored in temporary tree -->

   <xslt:template match="doc">
      <xslt:variable name="var1">
	        <xslt:element name="doc"> 
	           <xslt:attribute name="att">
	              <xslt:text>top</xslt:text>
	           </xslt:attribute>
	           <xslt:processing-instruction name="pi">PI_data</xslt:processing-instruction>
	           <xslt:comment>This is the 1st comment</xslt:comment>
	           <xslt:text>text-in-doc</xslt:text>
	           <xslt:element name="inner"/>
	        </xslt:element>
      </xslt:variable>
      <out>
         <t1>
            <xslt:value-of select="nilled($var1/doc/processing-instruction())"/>
         </t1>
         <t2>
            <xslt:value-of select="nilled($var1/doc/comment())"/>
         </t2>
         <t3>
            <xslt:value-of select="nilled($var1/doc/text()[1])"/>
         </t3>
         <t4>
            <xslt:value-of select="nilled($var1/doc/element())"/>
         </t4>
         <t5>
            <xslt:value-of select="nilled($var1/doc/attribute())"/>
         </t5>
         <t6>
            <xslt:for-each select="$var1/doc/namespace::*">
			            <xslt:value-of select="nilled(.)"/>
		          </xslt:for-each>
         </t6>
      </out>
   </xslt:template>
</xslt:transform>
