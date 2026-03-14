<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:string accessor with node from a temporary tree -->

   <xslt:template match="doc">
      <xslt:variable name="var1">
	        <xslt:element name="doc"> 
	           <xslt:namespace name="mynamespace" select="'http://test.com'"/>
	           <xslt:attribute name="att">
	              <xslt:text>top</xslt:text>
	           </xslt:attribute>
	           <xslt:processing-instruction name="pi">PI_data</xslt:processing-instruction>
	           <xslt:comment>This is the 1st comment</xslt:comment>
	           <xslt:text>text-in-doc</xslt:text>
	           <xslt:element name="inner">
               <xslt:text>text-in-doc2</xslt:text>	      
	           </xslt:element>
	        </xslt:element>
      </xslt:variable>
      <out>
         <t1>
            <xslt:value-of select="string($var1/doc)"/>
         </t1>
         <t2>
            <xslt:value-of select="string($var1/doc/inner)"/>
         </t2>
         <t3>
            <xslt:value-of select="string($var1/doc/attribute()[1])"/>
         </t3>
      </out>
   </xslt:template>
</xslt:transform>
